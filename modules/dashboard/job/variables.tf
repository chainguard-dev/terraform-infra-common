variable "job_name" {
  description = "Name of the job(s) to monitor"
  type        = string
}

variable "labels" {
  description = "Additional labels to apply to the dashboard."
  default     = {}
}

variable "project_id" {
  description = "ID of the GCP project"
  type        = string
}

variable "notification_channels" {
  description = "List of notification channels to alert."
  type        = list(string)
  default     = []
}
