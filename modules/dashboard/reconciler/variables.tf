/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

variable "project_id" {
  description = "The GCP project ID"
  type        = string
}

variable "name" {
  description = "The name of the reconciler (base name without suffixes)"
  type        = string
}

variable "service_name" {
  description = "The name of the reconciler service (defaults to name-rec)"
  type        = string
  default     = ""
}

variable "workqueue_name" {
  description = "The name of the workqueue (defaults to name-wq)"
  type        = string
  default     = ""
}

// Workqueue configuration
variable "max_retry" {
  description = "The maximum number of retry attempts for workqueue tasks"
  type        = number
  default     = 100
}

variable "concurrent_work" {
  description = "The amount of concurrent work the workqueue dispatches"
  type        = number
  default     = 20
}

// Section visibility
variable "sections" {
  description = "Configure visibility of optional dashboard sections"
  type = object({
    github = optional(bool, false)
  })
  default = {}
}

// Alert configuration
variable "alerts" {
  description = "Map of alert names to alert configurations"
  type = map(object({
    displayName         = string
    documentation       = string
    userLabels          = map(string)
    project             = string
    notificationChannel = string
  }))
  default = {}
}

variable "notification_channels" {
  description = "List of notification channels for alerts"
  type        = list(string)
  default     = []
}

variable "labels" {
  description = "Additional labels to add to the dashboard"
  type        = map(string)
  default     = {}
}
