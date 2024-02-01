variable "service_name" {
  description = "Name of the service(s) to monitor"
  type        = string
}

variable "labels" {
  description = "Additional labels to apply to the dashboard."
  default     = {}
}

variable "triggers" {
  description = "A mapping from a descriptive name to a subscription name prefix."
  type        = map(string)
}

variable "alert_threshold" {
  description = "The threshold for the alerting policy."
  type        = number
  default     = 50000
}

variable "notification_channels" {
  description = "The notification channels to use for the alerting policy."
  type        = list(string)
  default     = []
}
