/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

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

variable "shards" {
  description = "Number of workqueue shards (2-5). Each shard is an independent workqueue."
  type        = number
  default     = 2

  validation {
    condition     = var.shards >= 2 && var.shards <= 5
    error_message = "shards must be between 2 and 5"
  }
}

variable "concurrent-work" {
  description = "The amount of concurrent work to dispatch at a given time (distributed across shards)."
  type        = number
}

variable "batch-size" {
  description = "Optional cap on how much work to launch per dispatcher pass."
  type        = number
  default     = null
}

variable "max-retry" {
  description = "The maximum number of retry attempts before a task is moved to the dead letter queue."
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
  description = "Team label to apply to resources."
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

variable "multi_regional_location" {
  description = "The multi-regional location for the workqueue buckets."
  type        = string
  default     = "US"

  validation {
    condition     = contains(["US", "EU", "ASIA"], var.multi_regional_location)
    error_message = "multi_regional_location must be one of 'US', 'EU', or 'ASIA'."
  }
}
