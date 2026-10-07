/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

variable "project_id" {
  description = "The GCP project whose Personalized Service Health events are alerted on. Service Health is project-scoped: only incidents relevant to this project are reported."
  type        = string
}

variable "name" {
  description = "Optional environment name (e.g. enforce.dev) prefixed to the notification subject, so several environments can share a channel."
  type        = string
  default     = ""
}

variable "notification_channels_slack" {
  description = "Slack notification channels to notify, as projects/<project>/notificationChannels/<id>. Slack channels cannot be created through Terraform, so create them in the console and look them up with the google_monitoring_notification_channel data source."
  type        = list(string)
  default     = []
  nullable    = false

  validation {
    condition     = alltrue([for c in var.notification_channels_slack : can(regex("^projects/[^/]+/notificationChannels/[^/]+$", c))])
    error_message = "notification_channels_slack entries must be projects/<project>/notificationChannels/<id>."
  }
}

variable "notification_channels_email" {
  description = "Email notification channels to notify, as projects/<project>/notificationChannels/<id>."
  type        = list(string)
  default     = []
  nullable    = false

  validation {
    condition     = alltrue([for c in var.notification_channels_email : can(regex("^projects/[^/]+/notificationChannels/[^/]+$", c))])
    error_message = "notification_channels_email entries must be projects/<project>/notificationChannels/<id>."
  }
}

variable "notification_channels_pubsub" {
  description = "Pub/Sub notification channels to notify, as projects/<project>/notificationChannels/<id>. Useful for routing incidents to a custom formatter or another system."
  type        = list(string)
  default     = []
  nullable    = false

  validation {
    condition     = alltrue([for c in var.notification_channels_pubsub : can(regex("^projects/[^/]+/notificationChannels/[^/]+$", c))])
    error_message = "notification_channels_pubsub entries must be projects/<project>/notificationChannels/<id>."
  }
}

variable "notification_channels" {
  description = "Any other notification channels (e.g. webhook, PagerDuty, incident.io), as projects/<project>/notificationChannels/<id>."
  type        = list(string)
  default     = []
  nullable    = false

  validation {
    condition     = alltrue([for c in var.notification_channels : can(regex("^projects/[^/]+/notificationChannels/[^/]+$", c))])
    error_message = "notification_channels entries must be projects/<project>/notificationChannels/<id>."
  }
}

variable "products" {
  description = "Only alert on incidents impacting at least one of these products, matched as a substring of the event's impacted products (e.g. \"Cloud Run\", \"Google Kubernetes Engine\"). Empty alerts on every product."
  type        = list(string)
  default     = []
  nullable    = false
}

variable "locations" {
  description = "Only alert on incidents impacting at least one of these locations, matched as a substring of the event's impacted locations (e.g. \"us-central1\", \"global\"). Empty alerts on every location."
  type        = list(string)
  default     = []
  nullable    = false
}

variable "notification_rate_limit" {
  description = "Minimum time between notifications from the alert policy. Log-match policies notify per matching log entry, and the limit applies across all incidents, so a large value can delay updates for a second, concurrent incident."
  type        = string
  default     = "300s"
}

variable "auto_close" {
  description = "How long a Monitoring incident opened by the policy stays open before it auto-closes. Service Health posts its own RESOLVED update, so this only controls Monitoring incident bookkeeping."
  type        = string
  default     = "1800s"
}

variable "severity" {
  description = "Severity of the alert policy: CRITICAL, ERROR, or WARNING."
  type        = string
  default     = "WARNING"

  validation {
    condition     = contains(["CRITICAL", "ERROR", "WARNING"], var.severity)
    error_message = "severity must be one of CRITICAL, ERROR, or WARNING."
  }
}

variable "enable_api" {
  description = "Whether to enable servicehealth.googleapis.com in the project. Set to false if the API is enabled elsewhere."
  type        = bool
  default     = true
}

variable "team" {
  description = "Team label for the alert policy."
  type        = string
  default     = null
}
