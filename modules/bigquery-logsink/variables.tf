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

variable "partition_expiration_days" {
  description = "Global retention period in days for all table partitions"
  type        = number
  default     = 30
}

variable "tables" {
  description = <<-EOT
    Map of tables to create. Each key is the table name, and the value is an object with:
    - schema: JSON-encoded BigQuery schema
    - partition_field: Field name for time partitioning (required)
    - clustering_fields: List of fields for clustering (optional)
    - log_filter: Cloud Logging filter expression to route logs to this table
    - description: Table description (optional)
  EOT
  type = map(object({
    schema            = string
    partition_field   = string
    clustering_fields = optional(list(string), null)
    log_filter        = string
    description       = optional(string, "")
  }))
}

variable "deletion_protection" {
  description = "Enable deletion protection on tables"
  type        = bool
  default     = true
}

variable "use_partitioned_tables" {
  description = "Whether to use partitioned tables in log sink destinations"
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
