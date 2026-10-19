variable "project_id" {
  type = string
}

variable "notification_channels" {
  description = "List of notification channels to alert."
  type        = list(string)
  default     = []
}

variable "notification_channels_pager" {
  description = "Incident.io notification channel."
  type        = list(string)
  default     = []
}

variable "notification_channels_slack" {
  description = "Slack notification channel."
  type        = list(string)
  default     = []
}

variable "notification_channels_email" {
  description = "Email notification channel."
  type        = list(string)
  default     = []
}

variable "notification_channels_pubsub" {
  description = "Pubsub notification channel."
  type        = list(string)
  default     = []
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

variable "exitcode_filter" {
  description = "additional filter to apply to exitcode alert policy"
  type        = string
  default     = ""
}

variable "job_exitcode_filter" {
  description = "additional filter to apply to job exitcode alert policy"
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

variable "grpc_failure_rate_exclude_services" {
  description = "List of service names to exclude from non-grpc failure rate alert"
  type        = list(string)
  default     = []
}

variable "dlq_filter" {
  description = "additional filter to apply to dlq alert policy"
  type        = string
  default     = ""
}

variable "dlq_alert_threshold" {
  description = "Number of dead-letter queue messages above which the alert fires."
  type        = number
  default     = 1
}

variable "dlq_alert_duration" {
  description = "How long the dead-letter queue message count must stay above the threshold before the alert fires (e.g. '0s', '600s')."
  type        = string
  default     = "0s"
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

variable "team" {
  description = "Team label to apply to resources (replaces deprecated 'squad')."
  type        = string
}

variable "global_only_alerts" {
  description = "only enable global alerts. when true, only create alerts that are global."
  type        = bool
  default     = false
}

variable "http_error_threshold" {
  description = "threshold for http error."
  type        = number
  default     = 0.25
}

variable "http_error_exclude_services" {
  description = "List of service names to exclude from the http error rate alert."
  type        = list(string)
  default     = []
}

variable "http_error_method_status_exclusions" {
  description = "Map of service names to method/status exclusions. Each entry excludes the service from the main http error condition and generates replacement conditions that exclude the specified method+code combination(s). Set `method` for a single HTTP method or `methods` for several, and `code` for a single status code or `codes` for several; exactly one of each pair must be set."
  type = map(object({
    method  = optional(string)
    methods = optional(list(string), [])
    code    = optional(string)
    codes   = optional(list(string), [])
  }))
  default = {}

  validation {
    condition = alltrue([
      for k, v in var.http_error_method_status_exclusions : (v.code != null) != (length(v.codes) > 0)
    ])
    error_message = "Each exclusion must set exactly one of `code` or `codes`."
  }

  validation {
    condition = alltrue([
      for k, v in var.http_error_method_status_exclusions : (v.method != null) != (length(v.methods) > 0)
    ])
    error_message = "Each exclusion must set exactly one of `method` or `methods`."
  }

  # Several methods render as one regular expression, where an empty or
  # non-letter entry would match more series than the methods it names.
  validation {
    condition = alltrue([
      for k, v in var.http_error_method_status_exclusions : alltrue([
        for m in concat(v.method == null ? [] : [v.method], v.methods) : can(regex("^[A-Z]+$", m))
      ])
    ])
    error_message = "Each HTTP method must be uppercase letters only, such as \"GET\"."
  }
}

variable "http_error_documentation" {
  description = "Markdown appended to the http error rate alert's documentation, below the summary line the module writes. Use it to say what the alert means for the team's services and how to troubleshoot it, with links to logs, dashboards or runbooks."
  type        = string
  default     = ""

  validation {
    condition     = length(var.http_error_documentation) <= 8000
    error_message = "http_error_documentation may not exceed 8000 characters; Cloud Monitoring rejects alert documentation longer than 8192 characters, and the module prepends a summary line."
  }
}

variable "grpc_error_threshold" {
  description = "threshold for grpc error."
  type        = number
  default     = 0.25
}

variable "grpc_non_error_codes" {
  description = "List of grpc codes to not counted as error, case-sensitive."
  type        = list(string)
  default = [
    "OK",
    "Aborted",
    "AlreadyExists",
    "Canceled",
    "NotFound",
  ]
}

variable "product" {
  description = "Unused: cost attribution comes from Resource Manager tags, not labels. Kept so existing callers still plan."
  type        = string
  default     = null
}

variable "enable_high_retry" {
  description = "Whether to enable the workqueue high retry alert"
  type        = bool
  default     = false
}
