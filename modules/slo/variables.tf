variable "project_id" {
  description = "GCP project ID"
  type        = string
}

variable "service_name" {
  description = "Name of service to setup SLO for."
  type        = string
}

variable "service_type" {
  description = "Type of service to setup SLO for."
  type        = string
  default     = "CLOUD_RUN"
}

variable "regions" {
  description = "A map from region names to a network and subnetwork.  A service will be created in each region configured to egress the specified traffic via the specified subnetwork."
  type = map(object({
    network = string
    subnet  = string
  }))
}

variable "slo" {
  description = "Configuration for setting up SLO"
  type = object({
    enable          = optional(bool, false)
    enable_alerting = optional(bool, false)
    availability = optional(object(
      {
        multi_region_goal = optional(number, 0.999)
        per_region_goal   = optional(number, 0.999)
      }
    ), null)
  })
  default = null
}

variable "notification_channels" {
  description = "List of notification channels to alert."
  type        = list(string)
}
