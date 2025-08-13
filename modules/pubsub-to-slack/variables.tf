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
  description = "The GCP region to deploy the Cloud Run service"
  type        = string
  default     = "us-central1"
}

variable "network" {
  description = "The VPC network for the Cloud Run service"
  type        = string
}

variable "subnet" {
  description = "The VPC subnet for the Cloud Run service"
  type        = string
}

variable "slack_webhook_url" {
  description = "The Slack webhook URL for sending notifications"
  type        = string
  sensitive   = true
}

variable "slack_channel" {
  description = "The Slack channel to send messages to (e.g., '#alerts' or '@user')"
  type        = string
  default     = "#alerts"
}

variable "message_template" {
  description = <<-EOT
  Template for formatting messages sent to Slack. Use $${field_name} for JSON field substitution.
  Example: "Alert: $${budgetDisplayName} exceeded $${alertThresholdExceeded*100}% ($${costAmount} $${currencyCode})"
  EOT
  type        = string
  default     = "Notification: $${message}"
}

variable "image" {
  description = "The container image for the Pub/Sub to Slack bridge service"
  type        = string
}

variable "squad" {
  description = "The squad/team label to apply to resources"
  type        = string
  default     = ""
}

variable "product" {
  description = "The product label to apply to resources"
  type        = string
  default     = ""
}

variable "labels" {
  description = "Additional labels to apply to resources"
  type        = map(string)
  default     = {}
}

// Cloud Run configuration
variable "cpu_limit" {
  description = "CPU limit for the Cloud Run service"
  type        = string
  default     = "1000m"
}

variable "memory_limit" {
  description = "Memory limit for the Cloud Run service"
  type        = string
  default     = "512Mi"
}

variable "min_instances" {
  description = "Minimum number of Cloud Run instances"
  type        = number
  default     = 0
}

variable "max_instances" {
  description = "Maximum number of Cloud Run instances"
  type        = number
  default     = 10
}

variable "max_concurrency" {
  description = "Maximum concurrent requests per Cloud Run instance"
  type        = number
  default     = 1000
}

// Pub/Sub configuration
variable "ack_deadline_seconds" {
  description = "The maximum time a message can be outstanding before being redelivered"
  type        = number
  default     = 60
}

variable "message_retention_duration" {
  description = "How long unacknowledged messages are retained"
  type        = string
  default     = "604800s" // 7 days
}

variable "max_delivery_attempts" {
  description = "Maximum number of delivery attempts for dead letter queue"
  type        = number
  default     = 5
}

variable "min_retry_delay" {
  description = "Minimum retry delay for failed messages"
  type        = string
  default     = "10s"
}

variable "max_retry_delay" {
  description = "Maximum retry delay for failed messages"
  type        = string
  default     = "300s"
}
