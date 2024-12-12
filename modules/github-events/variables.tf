variable "project_id" {
  type = string
}

variable "name" {
  type = string
}

variable "regions" {
  description = "A map from region names to a network and subnetwork. The bucket must be in one of these regions."
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

variable "max_delivery_attempts" {
  description = "The maximum number of delivery attempts for any event."
  type        = number
  default     = 5
}

variable "notification_channels" {
  description = "List of notification channels to alert."
  type        = list(string)
}

variable "secret_version_adder" {
  type        = string
  description = "The user allowed to populate new webhook secret versions."
}

variable "service-ingress" {
  type        = string
  description = <<EOD
Which type of ingress traffic to accept for the service (see regional-go-service). Valid values are:

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

variable "require_squad" {
  description = "Whether to require squad variable to be specified"
  type        = bool
  default     = false
}

variable "squad" {
  description = "squad label to apply to the service."
  type        = string
  default     = ""

  validation {
    condition     = !var.require_squad || var.squad != ""
    error_message = "squad needs to specified or disable check by setting require_squad = false"
  }
}
