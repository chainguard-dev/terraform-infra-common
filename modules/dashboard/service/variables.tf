variable "service_name" {
  description = "Name of the service(s) to monitor"
  type        = string
}

variable "labels" {
  description = "Additional labels to apply to the dashboard."
  default     = {}
}

variable "alerts" {
  description = "Alerting policies to add to the dashboard."
  type        = list(string)
  default     = []
}

variable "project_id" {
  description = "ID of the GCP project"
  type        = string
}
