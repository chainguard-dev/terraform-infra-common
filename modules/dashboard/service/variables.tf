variable "service_name" {
  description = "Name of the service(s) to monitor"
  type        = string
}

variable "labels" {
  description = "Additional labels to apply to the dashboard."
  default     = {}
}

variable "alerts" {
  description = "A mapping from alerting policy names to the alert ids to add to the dashboard."
  type        = map(string)
  default     = {}
}

variable "project_id" {
  description = "ID of the GCP project"
  type        = string
}
