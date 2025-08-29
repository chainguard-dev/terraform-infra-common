variable "project_id" {
  type = string
}

variable "name" {
  type = string
}

variable "regions" {
  description = "A map from region names to a network and subnetwork.  A service will be created in each region configured to egress the specified traffic via the specified subnetwork."
  type = map(object({
    network = string
    subnet  = string
  }))
}

variable "concurrent-work" {
  description = "The amount of concurrent work to dispatch at a given time."
  type        = number
}

variable "max-retry" {
  description = "The maximum number of retry attempts before a task is moved to the dead letter queue. Set this to 0 to have unlimited retries."
  type        = number
  default     = 100
}

variable "reconciler-service" {
  description = "The name of the reconciler service that the workqueue will dispatch work to."
  type = object({
    name = string
  })
}

variable "squad" {
  description = "squad label to apply to the service."
  type        = string
}

variable "deletion_protection" {
  type        = bool
  description = "Whether to enable delete protection for the service."
  default     = true
}

variable "notification_channels" {
  description = "List of notification channels to alert."
  type        = list(string)
}

variable "labels" {
  description = "Labels to apply to the workqueue resources."
  type        = map(string)
  default     = {}
}

variable "product" {
  description = "Product label to apply to the service."
  type        = string
  default     = "unknown"
}

variable "scope" {
  description = "The scope of the workqueue: 'regional' for region-specific workqueues or 'global' for a single multi-regional workqueue."
  type        = string
  default     = "regional"

  validation {
    condition     = contains(["regional", "global"], var.scope)
    error_message = "scope must be either 'regional' or 'global'."
  }
}

variable "multi_regional_location" {
  description = "The multi-regional location for the global workqueue bucket (e.g., 'US', 'EU', 'ASIA'). Only used when scope='global'."
  type        = string
  default     = "US"

  validation {
    condition     = contains(["US", "EU", "ASIA"], var.multi_regional_location)
    error_message = "multi_regional_location must be one of 'US', 'EU', or 'ASIA'."
  }
}
