variable "project_id" {
  description = "The GCP project ID where resources will be created"
  type        = string
}

variable "name" {
  description = "Base name for the BigQuery resources"
  type        = string
}

variable "location" {
  description = "BigQuery dataset location"
  type        = string
  default     = "US"
}

variable "dataset_description" {
  description = "Description of the BigQuery dataset"
  type        = string
  default     = ""
}

variable "delete_contents_on_destroy" {
  description = "Whether to delete dataset contents when destroying the dataset"
  type        = bool
  default     = false
}

variable "retention_days" {
  description = "The number of days to retain data in BigQuery. Partitions older than this will be automatically deleted. Only applies when use_partitioned_tables is true."
  type        = number
  default     = 30
}

variable "sinks" {
  description = <<-EOT
    Map of log sinks to create. Each key is the sink name suffix, and the value is an object with:
    - log_filter: Cloud Logging filter expression to route logs
    - description: Sink description (optional)

    Note: Tables are auto-created by Cloud Logging based on log names.
    See: https://cloud.google.com/logging/docs/export/bigquery
  EOT
  type = map(object({
    log_filter  = string
    description = optional(string, "")
  }))
}

variable "use_partitioned_tables" {
  description = "Whether to use partitioned tables in log sink destinations. Must be true for partition expiration to work."
  type        = bool
  default     = true
}

variable "team" {
  description = "Team label for resources"
  type        = string
  default     = null
}

variable "product" {
  description = "Product label for resources"
  type        = string
  default     = null
}

variable "labels" {
  description = "Additional labels to apply to resources"
  type        = map(string)
  default     = {}
}

variable "enable_monitoring" {
  description = "Enable monitoring alert policies for log ingestion"
  type        = bool
  default     = false
}

variable "notification_channels" {
  description = "List of notification channel IDs for alerts"
  type        = list(string)
  default     = []
}

variable "alert_threshold_minutes" {
  description = "Minutes without log ingestion before triggering alert"
  type        = number
  default     = 1 * 24 * 60 // 1 day
}

variable "alert_auto_close_days" {
  description = "Days after which to auto-close resolved alerts"
  type        = number
  default     = 1
}

variable "resource_manager_tags" {
  description = "Resource Manager tags to bind to the log sink dataset, as tagKeys/<id> => tagValues/<id>."
  type        = map(string)
  default     = {}
  nullable    = false

  validation {
    condition = alltrue([
      for key, value in var.resource_manager_tags :
      can(regex("^tagKeys/[0-9]+$", key)) && can(regex("^tagValues/[0-9]+$", value))
    ])
    error_message = "resource_manager_tags keys must be tagKeys/<numeric-id> and values must be tagValues/<numeric-id>."
  }

  # Every BigQuery binding here tests this same input, so guard it once.
  # (Referencing var.location requires Terraform >= 1.9; see .terraform-version.)
  validation {
    condition     = length(var.resource_manager_tags) == 0 || can(regex("-", var.location))
    error_message = "resource_manager_tags cannot be applied to a multi-region BigQuery location (${var.location}) until hashicorp/terraform-provider-google#18254 is fixed. Use a single-region location, or leave resource_manager_tags empty."
  }
}
