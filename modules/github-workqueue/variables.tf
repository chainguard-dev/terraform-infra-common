/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

variable "project_id" {
  description = "The GCP project ID"
  type        = string
}

variable "name" {
  description = "The base name for resources"
  type        = string
}

variable "regions" {
  description = "A map of regions to launch services in (see regional-go-service module for format)"
  type = map(object({
    network = string
    subnet  = string
  }))
}

variable "notification_channels" {
  description = "List of notification channels for alerts"
  type        = list(string)
}

# Workqueue configuration
variable "workqueue" {
  description = "The workqueue to send events to"
  type = object({
    name = string
  })
}

# Service configuration
variable "resource_filter" {
  description = "Optional filter to process only specific resource types"
  type        = string
  default     = ""

  validation {
    condition     = var.resource_filter == "" || var.resource_filter == "issues" || var.resource_filter == "pull_requests"
    error_message = "Resource filter must be empty (no filter), 'issues', or 'pull_requests'."
  }
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
