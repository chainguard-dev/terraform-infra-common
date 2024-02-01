variable "service_name" {
  description = "Name of the service(s) to monitor"
  type        = string
}

variable "labels" {
  description = "Additional labels to apply to the dashboard."
  default     = {}
}

variable "triggers" {
  description = "A mapping from a descriptive name to a subscription name prefix and an alert threshold"
  type = map(object({
    subscription_prefix = string
    alert_threshold     = optional(number, 50000)
  }))
}

variable "notification_channels" {
  description = "The notification channels to use for the alerting policy."
  type        = list(string)
  default     = []
}
