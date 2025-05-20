/**
 * Copyright 2025 Chainguard, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = ">= 4.79"
    }
  }
}

locals {
  default_labels = {
    "redis" = var.name
  }

  squad_label = var.squad != "" ? {
    squad = var.squad
    team  = var.squad
  } : {}

  merged_labels = merge(local.default_labels, local.squad_label)
}

# Enable the Redis API
resource "google_project_service" "redis_api" {
  project                    = var.project_id
  service                    = "redis.googleapis.com"
  disable_dependent_services = false
}

resource "google_redis_instance" "default" {
  depends_on = [google_project_service.redis_api]

  name           = var.name
  project        = var.project_id
  region         = var.region
  location_id    = var.zone

  tier                    = var.tier
  memory_size_gb          = var.memory_size_gb
  redis_version           = var.redis_version
  reserved_ip_range       = var.reserved_ip_range

  connect_mode            = var.connect_mode
  auth_enabled            = var.auth_enabled
  transit_encryption_mode = var.transit_encryption_mode

  read_replicas_mode      = var.read_replicas_mode
  replica_count           = var.replica_count

  # Alternative location for HA setup
  alternative_location_id = var.alternative_location_id != "" ? var.alternative_location_id : null

  # Maintenance policy configuration
  dynamic "maintenance_policy" {
    for_each = var.maintenance_policy != null ? [var.maintenance_policy] : []
    content {
      weekly_maintenance_window {
        day = maintenance_policy.value.day
        start_time {
          hours   = maintenance_policy.value.start_time.hours
          minutes = maintenance_policy.value.start_time.minutes
          seconds = maintenance_policy.value.start_time.seconds
          nanos   = maintenance_policy.value.start_time.nanos
        }
      }
    }
  }

  # Network configuration
  authorized_network = var.authorized_network != "" ? var.authorized_network : null

  # Persistence configuration for backups
  persistence_config {
    persistence_mode    = var.persistence_config.persistence_mode
    rdb_snapshot_period = var.persistence_config.persistence_mode == "RDB" ? var.persistence_config.rdb_snapshot_period : null
  }

  labels = local.merged_labels
}

resource "google_project_iam_member" "redis_client_sa" {
  for_each = toset(var.authorized_client_service_accounts)

  project = var.project_id
  role    = "roles/redis.viewer"  # Read-only access by default
  member  = "serviceAccount:${each.value}"
}

resource "google_project_iam_member" "redis_editor_sa" {
  for_each = toset(var.authorized_client_editor_service_accounts)

  project = var.project_id
  role    = "roles/redis.editor"  # Read-write access
  member  = "serviceAccount:${each.value}"
}
