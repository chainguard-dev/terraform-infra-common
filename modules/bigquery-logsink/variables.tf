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
  default     = 180
}

variable "alert_auto_close_days" {
  description = "Days after which to auto-close resolved alerts"
  type        = number
  default     = 1
}
