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
