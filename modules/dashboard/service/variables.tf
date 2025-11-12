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

variable "notification_channels" {
  description = "List of notification channels to alert."
  type        = list(string)
}

variable "sections" {
  description = "Sections to include in the dashboard"
  type = object({
    http   = optional(bool, true)  // Include HTTP section
    grpc   = optional(bool, true)  // Include GRPC section
    github = optional(bool, false) // Include GitHub API section
    gorm   = optional(bool, false) // Include GORM section
    agents = optional(bool, false) // Include Agent metrics section
  })
  default = {
    http   = true
    grpc   = true
    github = false
    gorm   = false
    agents = false
  }
}
