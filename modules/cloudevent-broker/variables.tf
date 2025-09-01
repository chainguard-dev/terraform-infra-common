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



variable "squad" {
  description = "squad label to apply to the service."
  type        = string
  default     = "unknown"

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
