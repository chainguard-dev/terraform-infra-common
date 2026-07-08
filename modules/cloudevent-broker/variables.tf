variable "project_id" {
  type = string
}

variable "name" {
  type = string
}

variable "regions" {
  description = "A map from region names to a network and subnetwork.  A pub/sub topic and ingress service (publishing to the respective topic) will be created in each region, with the ingress service configured to egress all traffic via the specified subnetwork."
  type = map(object({
    network = string
    subnet  = string
  }))
}

variable "notification_channels" {
  description = "List of notification channels to alert."
  type        = list(string)
}

variable "scaling" {
  description = "The scaling configuration for the service."
  type = object({
    min_instances                    = optional(number, 0)
    max_instances                    = optional(number, 100)
    max_instance_request_concurrency = optional(number)
  })
  default = {}
}

variable "limits" {
  description = "Resource limits for the regional go service."
  type = object({
    cpu    = string
    memory = string
  })
  default = null
}

variable "enable_profiler" {
  type        = bool
  default     = false
  description = "Enable cloud profiler."
}



variable "team" {
  description = "Team label to apply to resources (replaces deprecated 'squad')."
  type        = string
}

variable "labels" {
  description = "Labels to apply to the broker resources."
  type        = map(string)
  default     = {}
}

variable "product" {
  description = "Product label to apply to the service."
  type        = string
  default     = "unknown"
}

variable "deletion_protection" {
  type        = bool
  description = "Whether to enable delete protection for the service."
  default     = true
}

variable "cpu_idle" {
  description = "Set to false for a region in order to use instance-based billing. Defaults to true."
  type        = map(bool)
  default     = {}
}

variable "dedicated_topics" {
  description = "Map from CloudEvent ce-type to config for routing that type onto its own dedicated per-region topic instead of the shared broker firehose. Consumers of a dedicated type must subscribe to the dedicated topic (see the `dedicated` output) rather than the shared broker. Defaults to empty: every event goes to the shared topic, unchanged. Set route=false to create the topic and grants without yet routing to it (so consumers can create their dedicated-topic subscriptions before the type is routed there, making cutover lossless); flip to route=true once those subscriptions exist."
  type = map(object({
    route = optional(bool, true)
    # Topic retention for seek/replay. Defaults to the shared topic's 600s;
    # raise it for a high-volume dedicated stream where a longer incident
    # replay window is worth the storage.
    message_retention_duration = optional(string, "600s")
  }))
  default = {}

  validation {
    // Keys become part of a Pub/Sub topic ID ("<name>-<type with dots as
    // dashes>-<region>"), so restrict them to characters that stay valid and
    // bound the length well under the 255-char topic-ID limit.
    condition = alltrue([
      for type in keys(var.dedicated_topics) :
      can(regex("^[a-zA-Z0-9]([a-zA-Z0-9._-]*[a-zA-Z0-9])?$", type)) && length(type) <= 100
    ])
    error_message = "dedicated_topics keys must be <=100 chars and contain only letters, digits, '.', '-', '_' (starting and ending alphanumeric)."
  }
}

variable "extra_publishers" {
  description = "Additional service account emails (without 'serviceAccount:' prefix) to grant roles/pubsub.publisher on each regional broker topic. Listed alongside the ingress SA in the authoritative IAM binding."
  type        = list(string)
  default     = []
}

variable "ingress" {
  description = "Which type of ingress traffic to accept for the broker ingress Cloud Run service. Defaults to INGRESS_TRAFFIC_INTERNAL_ONLY so existing consumers see no diff. Set to INGRESS_TRAFFIC_ALL only when the broker must be reachable from outside any VPC (e.g. a CI environment without VPC connectivity)."
  type        = string
  default     = "INGRESS_TRAFFIC_INTERNAL_ONLY"
  validation {
    condition = contains([
      "INGRESS_TRAFFIC_ALL",
      "INGRESS_TRAFFIC_INTERNAL_ONLY",
      "INGRESS_TRAFFIC_INTERNAL_LOAD_BALANCER",
    ], var.ingress)
    error_message = "ingress must be one of: INGRESS_TRAFFIC_ALL, INGRESS_TRAFFIC_INTERNAL_ONLY, INGRESS_TRAFFIC_INTERNAL_LOAD_BALANCER."
  }
}

variable "require_authenticated_invocations" {
  description = "When true, do not grant roles/run.invoker to allUsers on the broker ingress service, even when ingress is not INTERNAL_ONLY. Defaults to false to preserve existing behavior. Set to true alongside a non-internal ingress so the broker is reachable over the public internet but only invocable by callers explicitly granted run.invoker (e.g. a CI service account), rejecting unauthenticated requests."
  type        = bool
  default     = false
}
