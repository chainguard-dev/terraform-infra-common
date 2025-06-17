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
