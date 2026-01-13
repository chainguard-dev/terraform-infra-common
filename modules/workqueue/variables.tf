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

variable "primary-region" {
  description = "The primary region for single-homed resources like the reenqueue job. Defaults to the first region in the regions map."
  type        = string
  default     = null
}

variable "concurrent-work" {
  description = "The amount of concurrent work to dispatch at a given time."
  type        = number
}

variable "batch-size" {
  description = "Optional cap on how much work to launch per dispatcher pass. Defaults to ceil(concurrent-work / number of regions) when unset."
  type        = number
  default     = null
}

variable "max-retry" {
  description = "The maximum number of retry attempts before a task is moved to the dead letter queue. Set this to 0 to have unlimited retries."
  type        = number
  default     = 100
}

variable "enable_dead_letter_alerting" {
  description = "Whether to enable alerting for dead-lettered keys."
  type        = bool
  default     = true
}

variable "reconciler-service" {
  description = "The name of the reconciler service that the workqueue will dispatch work to."
  type = object({
    name = string
  })
}

variable "team" {
  description = "Team label to apply to resources (replaces deprecated 'squad')."
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
  description = "The scope of the workqueue. Must be 'global' for a single multi-regional workqueue."
  type        = string
  default     = "global"

  validation {
    condition     = var.scope == "global"
    error_message = "scope must be 'global'. Regional scope is no longer supported."
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

variable "cpu_idle" {
  description = "Set to false for a region in order to use instance-based billing. Defaults to true."
  type        = map(map(bool))
  default = {
    "dispatcher" = {}
    "receiver"   = {}
  }
}
