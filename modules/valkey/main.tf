/**
 * Copyright 2026 Chainguard, Inc.
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
      version = ">= 7.34.0"
    }
  }
}

locals {
  default_labels = {
    basename(abspath(path.module)) = var.name
    terraform-module               = basename(abspath(path.module))
  }

  squad_label = {
    squad = var.team
    team  = var.team
  }
  product_label = var.product != "" ? {
    product = var.product
  } : {}

  merged_labels = merge(local.default_labels, local.squad_label, local.product_label, var.labels)
}

resource "google_project_service" "memorystore" {
  project            = var.project_id
  service            = "memorystore.googleapis.com"
  disable_on_destroy = false
}

resource "google_memorystore_instance" "valkey" {
  project     = var.project_id
  instance_id = var.name
  location    = var.region

  engine_version = var.engine_version
  mode           = var.mode
  node_type      = var.node_type
  shard_count    = var.shard_count
  replica_count  = var.replica_count

  # Clients connect as their workload identity under
  # roles/memorystore.dbConnectionUser and pin the managed CA (the ca_pem
  # output).
  authorization_mode      = "IAM_AUTH"
  transit_encryption_mode = "SERVER_AUTHENTICATION"

  engine_configs = length(var.engine_configs) > 0 ? var.engine_configs : null

  deletion_protection_enabled = var.deletion_protection_enabled

  # Endpoint auto-creation requires a gcp-memorystore service connection
  # policy on the network for this region; the caller owns it (GCP allows one
  # per network, region, and service class, so it cannot be per-instance).
  desired_auto_created_endpoints {
    network    = var.network
    project_id = var.project_id
  }

  zone_distribution_config {
    mode = var.zone_distribution.mode
    zone = var.zone_distribution.zone
  }

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

  persistence_config {
    mode = var.persistence_config.mode
    dynamic "rdb_config" {
      for_each = var.persistence_config.mode == "RDB" ? [1] : []
      content {
        rdb_snapshot_period = var.persistence_config.rdb_snapshot_period
      }
    }
    dynamic "aof_config" {
      for_each = var.persistence_config.mode == "AOF" ? [1] : []
      content {
        append_fsync = var.persistence_config.aof_append_fsync
      }
    }
  }

  labels = local.merged_labels

  lifecycle {
    precondition {
      condition     = var.mode == "CLUSTER" || var.shard_count == 1
      error_message = "shard_count must be 1 when mode is CLUSTER_DISABLED; set mode = \"CLUSTER\" to shard."
    }
  }

  depends_on = [google_project_service.memorystore]
}

locals {
  connections = flatten([
    for ep in google_memorystore_instance.valkey.endpoints : [
      for conn in ep.connections : conn.psc_auto_connection
    ]
  ])

  # CLUSTER_DISABLED serves standalone clients at the primary endpoint; CLUSTER
  # serves cluster-protocol clients at the discovery endpoint. one()-pick the
  # expected connection so a topology that ever surfaces more than one fails
  # the apply instead of dialing the wrong endpoint.
  connection_type = var.mode == "CLUSTER_DISABLED" ? "CONNECTION_TYPE_PRIMARY" : "CONNECTION_TYPE_DISCOVERY"

  connection = one([
    for c in local.connections : c if c.connection_type == local.connection_type
  ])

  # A CLUSTER_DISABLED instance with replicas also carries a reader endpoint;
  # absent replicas (or in CLUSTER mode, where the discovery endpoint fronts
  # all nodes) this is null.
  reader = one([
    for c in local.connections : c if c.connection_type == "CONNECTION_TYPE_READER"
  ])
}

# Keyed by list index: client SA emails are commonly created in the same apply,
# and for_each keys must be known at plan time even when the values are not.
resource "google_project_iam_member" "db_connection_user" {
  for_each = { for i, sa in var.authorized_client_service_accounts : i => sa }

  project = var.project_id
  role    = "roles/memorystore.dbConnectionUser"
  member  = "serviceAccount:${each.value}"
}

# dbConnectionUser carries only memorystore.instances.connect; clients that
# resolve the connect endpoint and managed CA from the Memorystore API at boot
# also need memorystore.instances.get, which no narrower role carries.
resource "google_project_iam_member" "viewer" {
  for_each = { for i, sa in var.authorized_client_service_accounts : i => sa }

  project = var.project_id
  role    = "roles/memorystore.viewer"
  member  = "serviceAccount:${each.value}"
}
