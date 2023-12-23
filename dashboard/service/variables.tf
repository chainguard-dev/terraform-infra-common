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

# Currently our metrics does not have service_name label: we
# are working around by specifying the grpc_service name label
# instead while we fix the metric labeling.
variable "grpc_service_name" {
  description = "Name of the GRPC service(s) to monitor"
  type        = string
  default     = ""
}
