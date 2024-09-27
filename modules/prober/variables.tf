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

variable "base_image" {
  type        = string
  description = "The base image to use for the prober."
  default     = null
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

variable "regional-env" {
  default     = []
  description = "A list of object that provides a map env per region."
  type = list(object({
    name  = string
    value = map(string)
  }))
}

variable "secret_env" {
  default     = {}
  description = "A map of secrets to mount as environment variables from Google Secrets Manager (e.g. secret_key=secret_name)"
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
  description = "A list of notification channels to send alerts to."
  type        = list(string)
}

variable "enable_slo_alert" {
  type        = bool
  default     = false
  description = "If true, alert service availability dropping below SLO threshold. Outputs will return the alert ID for notification and dashboards."
}

variable "slo_threshold" {
  description = "The uptime percent required to meet the SLO for the service, expressed as a decimal in {0, 1}"
  type        = number
  default     = 0.999

  validation {
    condition     = var.slo_threshold >= 0 && var.slo_threshold <= 1
    error_message = "slo_threshold must be a decimal between 0 and 1"
  }
}

variable "slo_notification_channels" {
  description = "A list of notification channels to send alerts to."
  type        = list(string)
  default     = []
}

variable "slo_policy_link" {
  description = "An optional link to the SLO policy to include in the alert documentation."
  type        = string
  default     = ""
}

variable "enable_profiler" {
  type        = bool
  default     = false
  description = "Enable cloud profiler."
}

variable "scaling" {
  description = "The scaling configuration for the service."
  type = object({
    min_instances                    = optional(number, 0)
    max_instances                    = optional(number, 100)
    max_instance_request_concurrency = optional(number)
  })
  default = {}
}

variable "service_timeout_seconds" {
  description = "The timeout set on the cloud run service routing the uptime check request."
  type        = number
  default     = "300"
}

variable "selected_regions" {
  // https://cloud.google.com/monitoring/api/ref_v3/rest/v3/projects.uptimeCheckConfigs#UptimeCheckRegion
  description = "List of uptime check region, minimum 3. Valid [USA (has 3 regions), EUROPE, SOUTH_AMERICA, ASIA_PACIFIC, USA_OREGON, USA_IOWA, USA_VIRGINIA]"
  type        = list(string)
  default     = null
}
