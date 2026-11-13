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
  required_version = ">= 1.11"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = ">= 8.7.0"
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
  }
  merged_labels = merge(local.default_labels, local.squad_label, var.labels)

  # Published on the AUTH secret (see "Connecting without getAuthString" in
  # the README). The CA bundle carries every current server CA, so a client
  # trusting it keeps connecting through a CA rotation; each PEM block ends in
  # a newline so the blocks never run together. An empty value is
  # left out rather than sent, since an instance without TLS has no CA.
  connection_annotations = { for k, v in {
    "redis-host"          = google_redis_instance.default.host
    "redis-port"          = tostring(google_redis_instance.default.port)
    "redis-server-ca-pem" = join("", [for c in google_redis_instance.default.server_ca_certs : "${trimspace(c.cert)}\n"])
  } : k => v if v != "" }

  redis_configs = { for k, v in {
    "notify-keyspace-events" = var.notify_keyspace_events
    "maxmemory-policy"       = var.maxmemory_policy
  } : k => v if v != null }
}

# Enable the Redis API
resource "google_project_service" "redis_api" {
  project                    = var.project_id
  service                    = "redis.googleapis.com"
  disable_dependent_services = false
  disable_on_destroy         = false
}

resource "google_redis_instance" "default" {
  depends_on = [google_project_service.redis_api]

  name        = var.name
  project     = var.project_id
  region      = var.region
  location_id = var.zone

  tier              = var.tier
  memory_size_gb    = var.memory_size_gb
  redis_version     = var.redis_version
  reserved_ip_range = var.reserved_ip_range

  redis_configs = length(local.redis_configs) > 0 ? local.redis_configs : null

  connect_mode            = var.connect_mode
  auth_enabled            = var.auth_enabled
  transit_encryption_mode = var.transit_encryption_mode

  read_replicas_mode = var.read_replicas_mode
  # replica_count can only be set when read replicas are enabled. With read
  # replicas disabled, GCP rejects any replica_count in an update
  # (FAILED_PRECONDITION) and manages the value itself (0 on BASIC, 1 for the
  # STANDARD_HA standby), so leave it null and let the API own it.
  replica_count = var.read_replicas_mode == "READ_REPLICAS_ENABLED" ? var.replica_count : null

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
  role    = "roles/redis.viewer" # Read-only access by default
  member  = "serviceAccount:${each.value}"
}

resource "google_project_iam_member" "redis_editor_sa" {
  for_each = toset(var.authorized_client_editor_service_accounts)

  project = var.project_id
  role    = "roles/redis.editor" # Read-write access
  member  = "serviceAccount:${each.value}"
}

# Create Secret Manager secret for the auth string
module "redis_auth_secret" {
  source = "../secret"
  count  = var.auth_enabled ? 1 : 0

  project_id = var.project_id
  name       = "${var.name}-auth"

  authorized-adder      = var.secret_version_adder
  authorized-adders     = var.secret_version_adders
  notification-channels = var.notification_channels

  alert_documentation = <<-EOT
    Unexpected Secret Manager activity targeted the Redis AUTH credential for `${var.name}` in `${var.region}`. The caller or operation is outside the configured accessor/version-manager exclusions; this may indicate credential exposure or an unrecognized deployment identity.

    Open [Logs Explorer](https://console.cloud.google.com/logs/query;query=${urlencode("protoPayload.serviceName=\"secretmanager.googleapis.com\"\n(protoPayload.request.name:\"/secrets/${var.name}-auth/\" OR protoPayload.request.parent:\"/secrets/${var.name}-auth\")")}?project=${var.project_id}) and check the principal, method, user agent, result and timestamp against recent Terraform runs and service deployments. Terraform no longer reads the AUTH payload, so an AccessSecretVersion from a CI identity is unexpected. Version-adder exclusions do not cover reads.

    For expected activity, confirm the identity with the Redis service owner before correcting the accessor or version-manager configuration. For unexplained activity, escalate to Security and the owning service team, investigate the caller's IAM access, and coordinate credential rotation and client updates if exposure is confirmed. Do not retrieve or copy the AUTH value into logs or incident notes, or broaden exclusions just to silence the alert.
  EOT

  # Additional viewer/editor service accounts will need to access the
  # secret to retrieve the auth string to establish a connection
  service-accounts = concat(
    [var.secret_accessor_sa_email],
    var.authorized_client_service_accounts,
    var.authorized_client_editor_service_accounts,
  )

  # Connection details, so a root that only connects to the instance can read
  # them from the secret's metadata. A data "google_redis_instance" lookup
  # also returns the AUTH string, which needs redis.instances.getAuthString.
  annotations = local.connection_annotations

  create_placeholder_version = false

  team = var.team

  resource_manager_tags = var.resource_manager_tags
}

# Store the AUTH string GCP generates. secret_data_wo is write-only: it never
# enters state, and refreshing the version reads only its metadata, never its
# payload, so plans need no secretAccessor. The string is still read from the
# instance: the provider calls getAuthString on every refresh of an AUTH-enabled
# google_redis_instance and keeps it in state (see "AUTH and plan identities"
# in the README).
resource "google_secret_manager_secret_version" "auth" {
  count  = var.auth_enabled ? 1 : 0
  secret = module.redis_auth_secret[0].secret_id

  secret_data_wo = google_redis_instance.default.auth_string
  # A replaced instance has a new AUTH string and a new create_time, so the
  # change replaces this version with one holding the new string. The
  # attribute is a string from provider 8.x (7.x had a number defaulting to 0).
  secret_data_wo_version = google_redis_instance.default.create_time
}

# The previous version stored secret_data in state, and every refresh read its
# payload. Forget it rather than destroy it: it holds the live AUTH string
# until auth above adds a newer version, and workloads resolving "latest" move
# to that one.
removed {
  from = google_secret_manager_secret_version.auth_string

  lifecycle {
    destroy = false
  }
}
