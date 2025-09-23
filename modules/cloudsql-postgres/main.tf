terraform {
  required_providers {
    google = {
      source = "hashicorp/google"
    }
  }
}

# Labels

locals {
  default_labels = {
    basename(abspath(path.module)) = var.name
    terraform-module               = basename(abspath(path.module))
  }

  squad_label = var.squad != "" ? {
    squad = var.squad
    team  = var.squad
  } : {}
  product_label = var.product != "" ? {
    product = var.product
  } : {}

  merged_labels = merge(local.default_labels, local.squad_label, local.product_label, var.labels)
}

# Primary Cloud SQL instance

resource "google_sql_database_instance" "this" {
  name    = var.name
  project = var.project
  region  = var.region

  database_version    = var.database_version
  deletion_protection = var.deletion_protection

  settings {
    tier                  = var.tier
    availability_type     = var.enable_high_availability ? "REGIONAL" : "ZONAL"
    disk_type             = "PD_SSD"
    disk_size             = var.storage_gb
    disk_autoresize       = true
    disk_autoresize_limit = var.disk_autoresize_limit

    # Optional edition (ENTERPRISE or ENTERPRISE_PLUS). Null lets provider use
    # its default based on database_version.
    edition = var.edition

    ip_configuration {
      ipv4_enabled                                  = false
      private_network                               = var.network
      ssl_mode                                      = var.ssl_mode
      enable_private_path_for_google_cloud_services = var.enable_private_path_for_google_cloud_services

      # Private Service Connect (PSC) configuration
      dynamic "psc_config" {
        for_each = var.psc_enabled ? [1] : []
        content {
          psc_enabled               = true
          allowed_consumer_projects = var.psc_allowed_consumer_projects
        }
      }
    }

    backup_configuration {
      enabled                        = var.backup_enabled
      start_time                     = var.backup_start_time
      point_in_time_recovery_enabled = var.enable_point_in_time_recovery
    }

    maintenance_window {
      day  = var.maintenance_window_day
      hour = var.maintenance_window_hour
    }

    dynamic "location_preference" {
      for_each = var.primary_zone == null ? [] : [var.primary_zone]
      content {
        zone = location_preference.value
      }
    }

    dynamic "database_flags" {
      for_each = merge({
        "cloudsql.iam_authentication" = "on"
      }, var.database_flags)
      content {
        name  = database_flags.key
        value = database_flags.value
      }
    }

    user_labels = local.merged_labels
  }

  lifecycle {
    # When disk_autoresize is enabled we let Cloud SQL grow storage automatically.
    # Ignore manual size changes in Terraform to prevent overriding autosizing.
    ignore_changes = [
      settings[0].disk_size
    ]
  }
}

# Optional read replicas

resource "google_sql_database_instance" "replicas" {
  for_each = toset(var.read_replica_regions)

  name                 = "${var.name}-${each.value}"
  project              = var.project
  region               = each.value
  database_version     = var.database_version
  master_instance_name = google_sql_database_instance.this.name
  deletion_protection  = var.replicas_deletion_protection

  replica_configuration {
    failover_target = false
  }

  settings {
    tier                  = var.tier
    disk_autoresize_limit = var.disk_autoresize_limit

    # Ensure replicas use same edition as primary
    edition = var.edition

    # Apply same database flags to replicas as primary
    dynamic "database_flags" {
      for_each = merge({
        "cloudsql.iam_authentication" = "on"
      }, var.database_flags)
      content {
        name  = database_flags.key
        value = database_flags.value
      }
    }

    backup_configuration {
      enabled                        = false
      point_in_time_recovery_enabled = false
    }

    ip_configuration {
      ipv4_enabled                                  = false
      private_network                               = var.network
      ssl_mode                                      = var.ssl_mode
      enable_private_path_for_google_cloud_services = var.enable_private_path_for_google_cloud_services

      # Private Service Connect (PSC) configuration
      dynamic "psc_config" {
        for_each = var.psc_enabled ? [1] : []
        content {
          psc_enabled               = true
          allowed_consumer_projects = var.psc_allowed_consumer_projects
        }
      }
    }

    user_labels = merge(local.merged_labels, {
      replica = "true"
      region  = each.value
    })
  }

  lifecycle {
    # When disk_autoresize is enabled we let Cloud SQL grow storage automatically.
    # Ignore manual size changes in Terraform to prevent overriding autosizing.
    ignore_changes = [
      settings[0].disk_size
    ]
  }

  depends_on = [google_sql_database_instance.this]
}

# Data‑plane IAM for authorized GSAs

resource "google_project_iam_member" "client_sa" {
  for_each = toset(var.authorized_client_service_accounts)

  project = var.project
  role    = "roles/cloudsql.client"
  member  = "serviceAccount:${each.value}"
}
