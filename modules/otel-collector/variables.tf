variable "project_id" {
  type = string
}

variable "service_account" {
  type        = string
  description = "The service account as which the collector will run."
}

variable "otel_collector_image" {
  type        = string
  default     = "chainguard/opentelemetry-collector-contrib:latest"
  description = "The otel collector image to use as a base."
}
