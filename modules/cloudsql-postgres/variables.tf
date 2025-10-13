variable "name" {
  description = "Cloud SQL instance name (lowercase letters, numbers, and hyphens; up to 98 characters)."
  type        = string
}

variable "project" {
  description = "GCP project ID hosting the Cloud SQL instance."
  type        = string
}

variable "labels" {
  description = "The resource labels to represent user-provided metadata."
  type        = map(string)
  default     = {}
}

variable "region" {
  description = "GCP region for the primary instance."
  type        = string
}

variable "network" {
  description = "Self‑link or name of the VPC network used for private IP connectivity."
  type        = string
}

variable "ssl_mode" {
  description = "SSL mode for the Cloud SQL instance. Default is TRUSTED_CLIENT_CERTIFICATE_REQUIRED."
  type        = string
  default     = "TRUSTED_CLIENT_CERTIFICATE_REQUIRED"

  validation {
    condition = contains([
      "TRUSTED_CLIENT_CERTIFICATE_REQUIRED",
      "ALLOW_UNENCRYPTED_AND_ENCRYPTED",
      "ENCRYPTED_ONLY"
    ], upper(var.ssl_mode))

    error_message = "ssl_mode must be one of: TRUSTED_CLIENT_CERTIFICATE_REQUIRED, ALLOW_UNENCRYPTED_AND_ENCRYPTED, ENCRYPTED_ONLY."
  }
}

variable "team" {
  description = "Team label to apply to resources (replaces deprecated 'squad')."
  type        = string
  default     = ""

  validation {
    condition     = var.team != "" || var.squad != ""
    error_message = "Either 'team' or 'squad' must be specified. Please use 'team' as 'squad' is deprecated."
  }
}

variable "squad" {
  description = "DEPRECATED: Use 'team' instead. Squad label to apply to resources."
  type        = string
  default     = ""
}

# Engine & Capacity

variable "database_version" {
  description = "Cloud SQL engine version."
  type        = string
  default     = "POSTGRES_17"

  validation {
    condition     = can(regex("^POSTGRES_", var.database_version))
    error_message = "database_version must begin with \"POSTGRES_\" (e.g. POSTGRES_17)."
  }
}

# For Cloud SQL Enterprise Plus edition instances, Cloud SQL offers predefined machine types.
# For Cloud SQL Enterprise edition instances, Cloud SQL offers predefined and custom machine types.
# see https://cloud.google.com/sql/docs/postgres/create-instance#machine-types
variable "edition" {
  description = <<EOT
    Cloud SQL edition for the instance. Accepted values:
      • "ENTERPRISE"
      • "ENTERPRISE_PLUS"
  EOT
  type        = string
  default     = null

  validation {
    condition = var.edition != null && contains([
      "ENTERPRISE",
      "ENTERPRISE_PLUS"
    ], upper(var.edition))

    error_message = "edition must be explicitly set to ENTERPRISE or ENTERPRISE_PLUS."
  }
}

# https://cloud.google.com/sql/docs/postgres/machine-series-overview
variable "tier" {
  description = "Machine tier for the Cloud SQL instance."
  type        = string
  default     = "db-perf-optimized-N-16" # 16 vCPUs, 128GB RAM
}

variable "storage_gb" {
  description = "Initial SSD storage size in GB."
  type        = number
  default     = 256
}

variable "disk_autoresize_limit" {
  description = "Upper GB limit for automatic storage growth. Set to 0 for unlimited."
  type        = number
  default     = 4096
}

# Availability & Replication

variable "enable_high_availability" {
  description = "Enable regional high‑availability (REGIONAL availability_type)."
  type        = bool
  default     = false
}

variable "primary_zone" {
  description = "Optional zone for the primary instance."
  type        = string
  default     = null
}

variable "read_replica_regions" {
  description = "List of regions in which to create read replicas. Empty list for none."
  type        = list(string)
  default     = []
}

variable "replicas_deletion_protection" {
  description = "Enable deletion protection for read replicas."
  type        = bool
  default     = false
}

# Backup & Maintenance

variable "backup_enabled" {
  description = "Enable automated daily backups."
  type        = bool
  default     = true

  validation {
    condition = var.backup_enabled || (
      var.enable_high_availability == false && length(var.read_replica_regions) == 0
    )
    error_message = "backup_enabled must be true when high availability or read replicas are enabled."
  }
}

variable "enable_point_in_time_recovery" {
  description = "Enable point-in-time recovery (continuous WAL archiving)."
  type        = bool
  default     = true

  validation {
    condition = var.enable_point_in_time_recovery || (
      var.enable_high_availability == false && length(var.read_replica_regions) == 0
    )
    error_message = "enable_point_in_time_recovery must be true when high availability or read replicas are enabled."
  }
}

variable "backup_start_time" {
  description = "Start time for the backup window (UTC, HH:MM)."
  type        = string
  default     = "08:00"

  validation {
    condition     = can(regex("^([01][0-9]|2[0-3]):[0-5][0-9]$", var.backup_start_time))
    error_message = "backup_start_time must be in HH:MM 24‑hour format (UTC)."
  }
}

variable "maintenance_window_day" {
  description = "Day of week for maintenance window (1=Mon … 7=Sun, 0 for unspecified)."
  type        = number
  default     = 7

  validation {
    condition     = var.maintenance_window_day >= 0 && var.maintenance_window_day <= 7
    error_message = "maintenance_window_day must be between 0 and 7."
  }
}

variable "maintenance_window_hour" {
  description = "Hour (0‑23 UTC) for the maintenance window."
  type        = number
  default     = 5

  validation {
    condition     = var.maintenance_window_hour >= 0 && var.maintenance_window_hour <= 23
    error_message = "maintenance_window_hour must be between 0 and 23."
  }
}

# Advanced Configuration

variable "database_flags" {
  description = "Additional database flags to set on the instance."
  type        = map(string)
  default     = {}
}

variable "authorized_client_service_accounts" {
  description = "List of Google service account emails that require `roles/cloudsql.client`."
  type        = list(string)
  default     = []
}

# Lifecycle & Protection

variable "deletion_protection" {
  description = "Cloud SQL API deletion protection flag. When true, prevents instance deletion via the API."
  type        = bool
  default     = true
}

variable "product" {
  description = "Product label to apply to the service."
  type        = string
  default     = "unknown"
}

variable "enable_private_path_for_google_cloud_services" {
  description = "Enable access from Google Cloud services (e.g. BigQuery)."
  type        = bool
  default     = false
}

# Private Service Connect (PSC) configuration

variable "psc_enabled" {
  description = "Enable Private Service Connect (PSC) for cross-project access to the Cloud SQL instance."
  type        = bool
  default     = false
}

variable "psc_allowed_consumer_projects" {
  description = "List of project IDs allowed to connect to this Cloud SQL instance via PSC. Only used when psc_enabled is true."
  type        = list(string)
  default     = []
}
