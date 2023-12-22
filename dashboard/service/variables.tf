variable "service_name" {
  description = "Name of the service(s) to monitor"
  type        = string
}

variable "labels" {
  description = "Additional labels to apply to the dashboard."
  default     = {}
}

variable "alert" {
  description = "Alerting policies to add to the dashboard."
  type        = string
  default     = ""
}

