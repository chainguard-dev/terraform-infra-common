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
