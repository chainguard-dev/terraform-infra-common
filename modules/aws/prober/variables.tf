/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

variable "name" {
  type        = string
  description = "Name to prefix to created resources."
  validation {
    condition     = length(var.name) <= 14
    error_message = "The name must be 14 characters or less to accommodate the '-canary' suffix (AWS CloudWatch Synthetics canary names have a 21-character limit)."
  }
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

variable "ingress" {
  type        = string
  description = "Network ingress configuration. PUBLIC for internet access, PRIVATE for VPC only"
  default     = "PUBLIC"
  validation {
    condition     = contains(["PUBLIC", "PRIVATE"], var.ingress)
    error_message = "Ingress must be either PUBLIC or PRIVATE"
  }
}

variable "egress" {
  type        = string
  description = "Network egress configuration. DEFAULT for internet, VPC for private resources"
  default     = "DEFAULT"
  validation {
    condition     = contains(["DEFAULT", "VPC"], var.egress)
    error_message = "Egress must be either DEFAULT or VPC"
  }
}

variable "vpc_connector_arn" {
  type        = string
  default     = null
  description = "Optional VPC connector ARN for private resource access (required if egress is VPC)."
}

variable "env" {
  default     = {}
  description = "A map of custom environment variables (e.g. key=value)"
  type        = map(string)
}

variable "secret_env" {
  default     = {}
  description = "A map of secrets to mount as environment variables from AWS Secrets Manager or SSM Parameter Store (e.g. secret_key=secret_arn)"
  type        = map(string)
}

variable "timeout" {
  type        = string
  default     = "60s"
  description = "The timeout for the prober in seconds. Supported values 1-60s"
  validation {
    condition     = can(regex("^([1-9]|[1-5][0-9]|60)s$", var.timeout))
    error_message = "timeout must be between 1s and 60s (e.g., '30s', '60s')"
  }
}

variable "period" {
  type        = string
  default     = "300s"
  description = "The period for the prober in seconds. Supported values: 60s (1 minute), 300s (5 minutes), 600s (10 minutes), and 900s (15 minutes)"
  validation {
    condition     = contains(["60s", "300s", "600s", "900s"], var.period)
    error_message = "period must be one of: 60s (1 minute), 300s (5 minutes), 600s (10 minutes), 900s (15 minutes)"
  }
}

variable "cpu" {
  type        = number
  default     = 1024
  description = "The CPU units for the prober. Valid values: 256, 512, 1024, 2048, 4096"
  validation {
    condition     = contains([256, 512, 1024, 2048, 4096], var.cpu)
    error_message = "cpu must be one of: 256, 512, 1024, 2048, 4096"
  }
}

variable "memory" {
  type        = number
  default     = 2048
  description = "The memory in MB for the prober. Valid values: 512, 1024, 2048, 3072, 4096, 6144, 8192, 10240, 12288"
  validation {
    condition     = contains([512, 1024, 2048, 3072, 4096, 6144, 8192, 10240, 12288], var.memory)
    error_message = "memory must be one of: 512, 1024, 2048, 3072, 4096, 6144, 8192, 10240, 12288"
  }
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

variable "uptime_alert_duration" {
  type        = string
  default     = "600s"
  description = "Duration for uptime alert policy."
}

variable "notification_channels" {
  description = "A list of SNS topic ARNs to send alerts to."
  type        = list(string)
  default     = []
}

variable "enable_profiler" {
  type        = bool
  default     = false
  description = "Enable cloud profiler (AWS X-Ray)."
}

variable "scaling" {
  description = "The scaling configuration for the service."
  type = object({
    min_instances                    = optional(number, 1)
    max_instances                    = optional(number, 25)
    max_instance_request_concurrency = optional(number, 100)
  })
  default = {}
}

variable "team" {
  description = "Team label to apply to resources."
  type        = string
}

variable "product" {
  description = "Product label to apply to the service."
  type        = string
}

variable "tags" {
  description = "Additional tags to apply to resources"
  type        = map(string)
  default     = {}
}

variable "cloudwatch_synthetics_enabled" {
  type        = bool
  default     = true
  description = "Enable CloudWatch Synthetics canary for uptime monitoring."
}

variable "canary_runtime_version" {
  type        = string
  default     = "syn-nodejs-puppeteer-13.0"
  description = "CloudWatch Synthetics runtime version."
}

variable "canary_schedule" {
  type        = string
  default     = "rate(5 minutes)"
  description = "CloudWatch Synthetics canary schedule expression."
}

variable "start_canary" {
  type        = bool
  default     = true
  description = "Automatically start the canary after creation. Set to false to create the canary in a stopped state."
}

variable "alarm_comparison_operator" {
  type        = string
  default     = "LessThanThreshold"
  description = "The arithmetic operation to use when comparing the specified statistic and threshold. Valid values: GreaterThanOrEqualToThreshold, GreaterThanThreshold, LessThanThreshold, LessThanOrEqualToThreshold."
  validation {
    condition     = contains(["GreaterThanOrEqualToThreshold", "GreaterThanThreshold", "LessThanThreshold", "LessThanOrEqualToThreshold"], var.alarm_comparison_operator)
    error_message = "comparison_operator must be one of: GreaterThanOrEqualToThreshold, GreaterThanThreshold, LessThanThreshold, LessThanOrEqualToThreshold"
  }
}

variable "alarm_evaluation_periods" {
  type        = number
  default     = 2
  description = "The number of periods over which data is compared to the specified threshold."
}

variable "alarm_threshold" {
  type        = number
  default     = 90
  description = "The value against which the specified statistic is compared. For SuccessPercent, this is the percentage (0-100)."
}

variable "alarm_statistic" {
  type        = string
  default     = "Average"
  description = "The statistic to apply to the alarm's associated metric. Valid values: SampleCount, Average, Sum, Minimum, Maximum."
  validation {
    condition     = contains(["SampleCount", "Average", "Sum", "Minimum", "Maximum"], var.alarm_statistic)
    error_message = "statistic must be one of: SampleCount, Average, Sum, Minimum, Maximum"
  }
}

variable "alarm_treat_missing_data" {
  type        = string
  default     = "notBreaching"
  description = "How to handle missing data points. Valid values: missing, ignore, breaching, notBreaching."
  validation {
    condition     = contains(["missing", "ignore", "breaching", "notBreaching"], var.alarm_treat_missing_data)
    error_message = "treat_missing_data must be one of: missing, ignore, breaching, notBreaching"
  }
}

variable "alarm_datapoints_to_alarm" {
  type        = number
  default     = null
  description = "The number of datapoints that must be breaching to trigger the alarm. Defaults to evaluation_periods if not set."
}

variable "create_instance_role" {
  type        = bool
  description = "Whether to create the IAM instance role for the running containers. If false, you must provide instance_role_arn."
  default     = true
}

variable "instance_role_arn" {
  type        = string
  description = "The ARN of the IAM role that the running service will assume. Required if create_instance_role is false."
  default     = ""
}
