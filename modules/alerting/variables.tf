variable "project_id" {
  type = string
}

variable "notification_channels" {
  description = "List of notification channels to alert."
  type        = list(string)
  default     = []
}

variable "notification_channel_pagerduty" {
  description = "Email notification channel."
  type        = string
  default     = ""
}

variable "notification_channel_slack" {
  description = "Slack notification channel."
  type        = string
  default     = ""
}

variable "notification_channel_email" {
  description = "Email notification channel."
  type        = string
  default     = ""
}

variable "oom_filter" {
  description = "additional filter to apply to oom alert policy"
  type        = string
  default     = ""
}

variable "signal_filter" {
  description = "additional filter to apply to signal alert policy"
  type        = string
  default     = ""
}

variable "failed_req_filter" {
  description = "additional filter to apply to failed request alert policy"
  type        = string
  default     = ""
}

variable "scaling_issue_filter" {
  description = "additional filter to apply to scaling issue alert policy"
  type        = string
  default     = ""
}

variable "failure_rate_ratio_threshold" {
  description = "ratio threshold to alert for cloud run server failure rate."
  type        = number
  default     = 0.2
}

variable "failure_rate_duration" {
  description = "duration for condition to be active before alerting"
  type        = number
  default     = 120
}

variable "failure_rate_exclude_services" {
  description = "List of service names to exclude from the 5xx failure rate alert"
  type        = list(string)
  default     = []
}

variable "dlq_filter" {
  description = "additional filter to apply to dlq alert policy"
  type        = string
  default     = ""
}

variable "panic_filter" {
  description = "additional filter to apply to panic alert policy"
  type        = string
  default     = ""
}

variable "timeout_filter" {
  description = "additional filter to apply to timeout alert policy"
  type        = string
  default     = ""
}

variable "enable_scaling_alerts" {
  description = <<EOT
  Whether to enable scaling alerts.
  When logs appear with
    "The request was aborted because there was no available instance." or
    "The request failed because either the HTTP response was malformed or connection to the instance had an error."
EOT
  type        = bool
  default     = false
}
