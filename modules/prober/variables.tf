/*
Copyright 2022 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

variable "name" {
  type        = string
  description = "Name to prefix to created resources."
}

variable "project_id" {
  type        = string
  description = "The project that will host the prober."
}

variable "service_account" {
  type        = string
  description = "The email address of the service account to run the service as."
}

variable "importpath" {
  type        = string
  description = "The import path that contains the prober application."
}

variable "working_dir" {
  type        = string
  description = "The working directory that contains the importpath."
}

variable "egress" {
  type        = string
  description = "The level of egress the prober requires."
  default     = "ALL_TRAFFIC"
}

variable "regions" {
  description = "A map from region names to a network and subnetwork.  A prober service will be created in each region."
  type = map(object({
    network = string
    subnet  = string
  }))
}

variable "dns_zone" {
  type        = string
  default     = ""
  description = "The managed DNS zone in which to create prober record sets (required for multiple locations)."
}

variable "domain" {
  type        = string
  default     = ""
  description = "The domain of the environment to probe (required for multiple locations)."
}

variable "env" {
  default     = {}
  description = "A map of custom environment variables (e.g. key=value)"
}

variable "timeout" {
  type        = string
  default     = "60s"
  description = "The timeout for the prober in seconds."
}

variable "period" {
  type        = string
  default     = "300s"
  description = "The period for the prober in seconds."
}

variable "cpu" {
  type        = string
  default     = "1000m"
  description = "The CPU limit for the prober."
}

variable "memory" {
  type        = string
  default     = "512Mi"
  description = "The memory limit for the prober."
}

variable "enable_alert" {
  type        = bool
  default     = false
  description = "If true, alert on failures. Outputs will return the alert ID for notification and dashboards."
}

variable "alert_description" {
  type        = string
  default     = "An uptime check has failed."
  description = "Alert documentation. Use this to link to playbooks or give additional context."
}

variable "notification_channels" {
  type        = list(string)
  default     = []
  description = "A list of notification channels to send alerts to."
}
