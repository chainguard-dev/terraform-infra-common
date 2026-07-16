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

variable "project_id" {
  description = "The ID of the project in which the resource belongs."
  type        = string
}

variable "name" {
  description = "The instance ID, also used to name the service connection policy."
  type        = string
}

variable "region" {
  description = "The GCP region to deploy resources to."
  type        = string
}

variable "network" {
  description = "The VPC network (id or self link) the instance's PSC endpoints are created in. The network must already carry a gcp-memorystore service connection policy for the instance's region (see the README's prerequisites); GCP allows one per (network, region, service class), so the policy is caller-owned, not per-instance."
  type        = string
}

variable "engine_version" {
  description = "The version of Valkey software, e.g. VALKEY_9_0."
  type        = string
  default     = "VALKEY_9_0"

  validation {
    condition     = startswith(var.engine_version, "VALKEY_")
    error_message = "engine_version must be a VALKEY_* version; this module does not support the Redis engine."
  }
}

variable "mode" {
  description = "The instance mode. CLUSTER_DISABLED serves standalone clients at a single primary endpoint; CLUSTER serves cluster-protocol clients at a discovery endpoint."
  type        = string
  default     = "CLUSTER_DISABLED"

  validation {
    condition     = contains(["CLUSTER", "CLUSTER_DISABLED"], var.mode)
    error_message = "mode must be either CLUSTER or CLUSTER_DISABLED."
  }
}

variable "node_type" {
  description = "The machine type of each node."
  type        = string
  default     = "STANDARD_SMALL"

  validation {
    condition     = contains(["SHARED_CORE_NANO", "STANDARD_SMALL", "HIGHMEM_MEDIUM", "HIGHMEM_XLARGE"], var.node_type)
    error_message = "node_type must be one of: SHARED_CORE_NANO, STANDARD_SMALL, HIGHMEM_MEDIUM, HIGHMEM_XLARGE."
  }
}

variable "shard_count" {
  description = "The number of shards. Must be 1 when mode is CLUSTER_DISABLED."
  type        = number
  default     = 1

  validation {
    condition     = var.shard_count >= 1 && floor(var.shard_count) == var.shard_count
    error_message = "shard_count must be a whole number >= 1."
  }
}

variable "zone_distribution" {
  description = "Zone distribution of the instance's nodes. MULTI_ZONE spreads nodes for availability; SINGLE_ZONE places all nodes in the given zone, co-locating with zonal clients to cut cross-zone latency and egress."
  type = object({
    mode = string
    zone = optional(string)
  })
  default = {
    mode = "MULTI_ZONE"
  }

  validation {
    condition     = contains(["MULTI_ZONE", "SINGLE_ZONE"], var.zone_distribution.mode)
    error_message = "zone_distribution.mode must be either MULTI_ZONE or SINGLE_ZONE."
  }

  validation {
    condition     = var.zone_distribution.mode != "SINGLE_ZONE" || var.zone_distribution.zone != null
    error_message = "zone_distribution.zone is required when mode is SINGLE_ZONE."
  }
}

variable "replica_count" {
  description = "The number of replica nodes per shard."
  type        = number
  default     = 1

  validation {
    condition     = var.replica_count >= 0 && var.replica_count <= 5 && floor(var.replica_count) == var.replica_count
    error_message = "replica_count must be a whole number between 0 and 5."
  }
}

variable "engine_configs" {
  description = "Engine configuration parameters, e.g. { maxmemory-policy = \"noeviction\" }."
  type        = map(string)
  default     = {}
}

variable "deletion_protection_enabled" {
  description = "Whether the instance refuses deletion until this is unset."
  type        = bool
  default     = true
}

variable "authorized_client_service_accounts" {
  description = "Service account emails granted roles/memorystore.dbConnectionUser (the IAM-auth connect grant) and roles/memorystore.viewer (instance metadata, for clients that resolve the endpoint and managed CA from the API at boot). Note these are project-level bindings: an authorized account can connect to any Memorystore instance in the project — so each SA must be listed by exactly one valkey module instance per project, or destroying one instance strips the grant the others rely on. Treat the list as append-only: entries are keyed by index, and removing or reordering a mid-list entry replaces the shifted grants, which can transiently (or, on an unlucky apply ordering, durably until re-apply) derole a surviving SA."
  type        = list(string)
  default     = []
}

variable "maintenance_policy" {
  description = "Maintenance policy for an instance."
  type = object({
    day = string
    start_time = object({
      hours   = optional(number, 0)
      minutes = optional(number, 0)
      seconds = optional(number, 0)
      nanos   = optional(number, 0)
    })
  })
  default = null
}

variable "persistence_config" {
  description = "Configuration of the persistence functionality."
  type = object({
    mode                = string
    rdb_snapshot_period = optional(string)
    aof_append_fsync    = optional(string)
  })
  default = {
    mode = "DISABLED"
  }

  validation {
    condition     = contains(["DISABLED", "RDB", "AOF"], var.persistence_config.mode)
    error_message = "Persistence mode must be one of: DISABLED, RDB, AOF."
  }

  validation {
    condition = (
      var.persistence_config.mode != "RDB" ||
      contains(["ONE_HOUR", "SIX_HOURS", "TWELVE_HOURS", "TWENTY_FOUR_HOURS"], coalesce(var.persistence_config.rdb_snapshot_period, "unset"))
    )
    error_message = "When mode is RDB, rdb_snapshot_period must be one of: ONE_HOUR, SIX_HOURS, TWELVE_HOURS, TWENTY_FOUR_HOURS."
  }

  validation {
    condition = (
      var.persistence_config.mode != "AOF" ||
      contains(["ALWAYS", "EVERY_SEC", "NEVER"], coalesce(var.persistence_config.aof_append_fsync, "unset"))
    )
    error_message = "When mode is AOF, aof_append_fsync must be one of: ALWAYS, EVERY_SEC, NEVER."
  }
}

variable "labels" {
  description = "The resource labels to represent user-provided metadata."
  type        = map(string)
  default     = {}
}

variable "team" {
  description = "Team label to apply to resources (replaces deprecated 'squad')."
  type        = string
}

variable "product" {
  description = "Product label to apply to the service."
  type        = string
  default     = "unknown"
}
