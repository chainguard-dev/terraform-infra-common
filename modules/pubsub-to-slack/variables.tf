/*
Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

variable "project_id" {
  description = "The GCP project ID"
  type        = string
}

variable "name" {
  description = "Name for the Pub/Sub to Slack bridge resources"
  type        = string
}

variable "region" {
  description = "The region where the service will be deployed"
  type        = string
}

variable "network" {
  description = "The network for the service to egress traffic via"
  type        = string
}

variable "subnet" {
  description = "The subnetwork for the service to egress traffic via"
  type        = string
}

variable "slack_webhook_secret_id" {
  description = "The Secret Manager secret ID containing the Slack webhook URL"
  type        = string
}

variable "slack_channel" {
  description = "The Slack channel to send messages to (e.g., '#alerts' or '@user')"
  type        = string
  default     = "#alerts"
}

variable "message_template" {
  description = "Template for formatting messages sent to Slack using Go template syntax. Use {{.field_name}} for JSON field substitution."
  type        = string
  default     = "Notification: {{.message}}"
}

variable "team" {
  description = "The team label to apply to resources"
  type        = string
  default     = "unknown"
}

variable "product" {
  description = "The product label to apply to resources"
  type        = string
  default     = "unknown"
}

variable "labels" {
  description = "Additional labels to apply to resources"
  type        = map(string)
  default     = {}
}

variable "notification_channels" {
  description = "List of notification channels to alert"
  type        = list(string)
  default     = []
}


variable "enable_profiler" {
  description = "Enable Cloud Profiler for the service"
  type        = bool
  default     = false
}
