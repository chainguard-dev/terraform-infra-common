variable "service_name" {
  description = "Name of the service(s) to monitor"
  type        = string
}

variable "labels" {
  description = "Additional labels to apply to the dashboard."
  default     = {}
}

variable "triggers" {
  description = "A mapping from a descriptive name to a subscription name prefix, an alert threshold, and list of notification channels."
  type = map(object({
    subscription_prefix   = string
    alert_threshold       = optional(number, 50000)
    notification_channels = optional(list(string), [])
  }))
}

variable "project_id" {
  description = "ID of the GCP project"
  type        = string
}

variable "notification_channels" {
  description = "List of notification channels to alert."
  type        = list(string)
}

variable "alerts" {
  description = "A mapping from alerting policy names to the alert ids to add to the dashboard."
  type        = map(string)
  default     = {}
}
