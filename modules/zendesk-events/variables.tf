variable "project_id" {
  type = string
}

variable "name" {
  type = string
}

variable "regions" {
  description = "A map from region names to a network and subnetwork."
  type = map(object({
    network = string
    subnet  = string
  }))
}

variable "deletion_protection" {
  type        = bool
  description = "Whether to enable delete protection for the service."
  default     = true
}

variable "ingress" {
  description = "An object holding the name of the ingress service, which can be used to authorize callers to publish cloud events."
  type = object({
    name = string
  })
}

variable "notification_channels" {
  description = "List of notification channels to alert."
  type        = list(string)
}

variable "secret_version_adder" {
  type        = string
  description = "The user allowed to populate new webhook secret versions."
}

variable "additional_webhook_secrets" {
  type = map(object({
    secret  = string
    version = string
  }))
  description = "Additional secrets to be used by the service."
  default     = {}
}

variable "service-ingress" {
  type        = string
  description = <<EOD
Which type of ingress traffic to accept for the service. Valid values are:

- INGRESS_TRAFFIC_ALL accepts all traffic, enabling the public .run.app URL for the service
- INGRESS_TRAFFIC_INTERNAL_LOAD_BALANCER accepts traffic only from a load balancer
EOD
  default     = "INGRESS_TRAFFIC_INTERNAL_LOAD_BALANCER"
}

variable "enable_profiler" {
  type        = bool
  default     = false
  description = "Enable cloud profiler."
}

variable "team" {
  description = "Team label to apply to resources."
  type        = string
}

variable "product" {
  description = "Product label to apply to the service."
  type        = string
  default     = "unknown"
}

variable "provisioner" {
  description = "The member-style identity of the account provisioning resources in this environment (e.g. serviceAccount:…). When set, it is granted access to the webhook secret so placeholder versions can be created."
  type        = string
  default     = ""
}

variable "create_placeholder_version" {
  type        = bool
  description = "Whether to create a placeholder secret version to avoid bad reference on first deploy."
  default     = false
}
