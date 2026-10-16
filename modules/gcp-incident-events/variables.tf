variable "project_id" {
  type = string
}

variable "name" {
  type = string
}

variable "topic" {
  description = "The ID (projects/<project>/topics/<topic>) of the Pub/Sub topic that a Cloud Monitoring pubsub notification channel publishes incidents to. The module creates its own subscription on it, so existing subscribers are unaffected."
  type        = string
}

variable "region" {
  description = "The region to run the trampoline service in. Must be a key of the regions variable."
  type        = string

  validation {
    condition     = contains(keys(var.regions), var.region)
    error_message = "region must be a key of regions."
  }
}

variable "regions" {
  description = "A map from region names to a network and subnetwork. Only the entry for the region variable is used."
  type = map(object({
    network = string
    subnet  = string
  }))
}

variable "ingress" {
  description = "An object holding the name of the ingress service, which can be used to authorize callers to publish cloud events."
  type = object({
    name = string
  })
}

variable "max_delivery_attempts" {
  description = "The maximum number of delivery attempts for any notification before it is dead-lettered."
  type        = number
  default     = 5
}

variable "notification_channels" {
  description = "Notification channels for this module's alerts on dead-lettered, stuck and failing incident deliveries. Must not include a pubsub notification channel that publishes to topic: an alert that forwarding is broken can't be delivered through the forwarder it reports on."
  type        = list(string)
}

variable "enable_profiler" {
  type        = bool
  default     = false
  description = "Enable cloud profiler."
}

variable "deletion_protection" {
  type        = bool
  description = "Whether to enable delete protection for the service."
  default     = true
}

variable "team" {
  description = "Team label to apply to resources."
  type        = string
}

variable "labels" {
  description = "Labels to apply to the incident event resources."
  type        = map(string)
  default     = {}
}

variable "product" {
  description = "Product label to apply to the service."
  type        = string
  default     = ""
}

variable "observability_role" {
  type        = string
  default     = null
  description = "Fully-qualified id of a single role (e.g. from the observability-role module) to grant the service account in place of the three built-in observability roles (monitoring.metricWriter, cloudtrace.agent, cloudprofiler.agent). Collapsing to one role keeps large projects under the 1,500-member IAM policy limit."

  validation {
    condition     = var.observability_role == null || can(regex("^(projects|organizations)/[^/]+/roles/[^/]+$", var.observability_role))
    error_message = "observability_role must be a fully-qualified role id: projects/{project}/roles/{role_id} or organizations/{org}/roles/{role_id}."
  }
}

variable "resource_manager_tags" {
  description = "Resource Manager tags to bind to this module's Pub/Sub resources, as tagKeys/<id> => tagValues/<id>."
  type        = map(string)
  default     = {}
  nullable    = false

  validation {
    condition = alltrue([
      for key, value in var.resource_manager_tags :
      can(regex("^tagKeys/[0-9]+$", key)) && can(regex("^tagValues/[0-9]+$", value))
    ])
    error_message = "resource_manager_tags keys must be tagKeys/<numeric-id> and values must be tagValues/<numeric-id>."
  }
}
