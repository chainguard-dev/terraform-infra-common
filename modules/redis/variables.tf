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

variable "project_id" {
  description = "The ID of the project in which the resource belongs."
  type        = string
}

variable "name" {
  description = "The ID of the instance or a fully qualified identifier for the instance."
  type        = string
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

variable "region" {
  description = "The GCP region to deploy resources to."
  type        = string
}

variable "zone" {
  description = "The zone where the instance will be deployed."
  type        = string
}

variable "alternative_location_id" {
  description = "The alternative zone where the instance will failover when zone is unavailable."
  type        = string
  default     = ""
}

variable "tier" {
  description = "The service tier of the instance. Valid values: BASIC, STANDARD_HA."
  type        = string
  default     = "STANDARD_HA"

  validation {
    condition     = contains(["BASIC", "STANDARD_HA"], var.tier)
    error_message = "The tier must be either BASIC or STANDARD_HA."
  }
}

variable "memory_size_gb" {
  description = "Redis memory size in GiB. Minimum 1 GB, maximum 300 GB."
  type        = number
  default     = 1

  validation {
    condition     = var.memory_size_gb >= 1 && var.memory_size_gb <= 300
    error_message = "Memory size must be between 1 and 300 GB."
  }
}

variable "replica_count" {
  description = "The number of replica nodes."
  type        = number
  default     = 0
}

variable "read_replicas_mode" {
  description = "Read replicas mode. Can be: READ_REPLICAS_DISABLED or READ_REPLICAS_ENABLED."
  type        = string
  default     = "READ_REPLICAS_DISABLED"

  validation {
    condition     = contains(["READ_REPLICAS_DISABLED", "READ_REPLICAS_ENABLED"], var.read_replicas_mode)
    error_message = "Read replicas mode must be either READ_REPLICAS_DISABLED or READ_REPLICAS_ENABLED."
  }
}

variable "redis_version" {
  description = "The version of Redis software."
  type        = string
  default     = "REDIS_7_2"
}

variable "reserved_ip_range" {
  description = "The CIDR range of internal addresses that are reserved for this instance."
  type        = string
  default     = null
}

variable "connect_mode" {
  description = "The connection mode of the Redis instance. Valid values: DIRECT_PEERING, PRIVATE_SERVICE_ACCESS."
  type        = string
  default     = "PRIVATE_SERVICE_ACCESS"

  validation {
    condition     = contains(["DIRECT_PEERING", "PRIVATE_SERVICE_ACCESS"], var.connect_mode)
    error_message = "Connect mode must be either DIRECT_PEERING or PRIVATE_SERVICE_ACCESS."
  }
}

variable "authorized_network" {
  description = "The full name of the Google Compute Engine network to which the instance is connected. Must be in the format: projects/{project_id}/global/networks/{network_name}"
  type        = string
  default     = ""
}

variable "maintenance_policy" {
  description = "Maintenance policy for an instance."
  type = object({
    day = string
    start_time = object({
      hours   = number
      minutes = number
      seconds = number
      nanos   = number
    })
  })
  default = null
}

variable "persistence_config" {
  description = "Configuration of the persistence functionality."
  type = object({
    persistence_mode    = string
    rdb_snapshot_period = optional(string)
  })
  default = {
    persistence_mode = "DISABLED"
  }

  # Check that persistence_mode is valid
  validation {
    condition     = contains(["DISABLED", "RDB"], var.persistence_config.persistence_mode)
    error_message = "Persistence mode must be either DISABLED or RDB."
  }

  # Check that we have valid configurations
  validation {
    condition = (
      var.persistence_config.persistence_mode == "DISABLED" ||

      (var.persistence_config.persistence_mode == "RDB" && var.persistence_config.rdb_snapshot_period == "ONE_HOUR") ||
      (var.persistence_config.persistence_mode == "RDB" && var.persistence_config.rdb_snapshot_period == "SIX_HOURS") ||
      (var.persistence_config.persistence_mode == "RDB" && var.persistence_config.rdb_snapshot_period == "TWELVE_HOURS") ||
      (var.persistence_config.persistence_mode == "RDB" && var.persistence_config.rdb_snapshot_period == "TWENTY_FOUR_HOURS")
    )
    error_message = "When persistence_mode is RDB, rdb_snapshot_period must be one of: ONE_HOUR, SIX_HOURS, TWELVE_HOURS, or TWENTY_FOUR_HOURS."
  }
}

variable "auth_enabled" {
  description = "Indicates whether AUTH is enabled for the instance."
  type        = bool
  default     = true
}

variable "transit_encryption_mode" {
  description = "The TLS mode of the Redis instance. Valid values: DISABLED, SERVER_AUTHENTICATION."
  type        = string
  default     = "SERVER_AUTHENTICATION"

  validation {
    condition     = contains(["DISABLED", "SERVER_AUTHENTICATION"], var.transit_encryption_mode)
    error_message = "Transit encryption mode must be either DISABLED or SERVER_AUTHENTICATION."
  }
}

variable "authorized_client_service_accounts" {
  description = "List of service account emails that should be granted Redis viewer (read-only) access"
  type        = list(string)
  default     = []
}

variable "authorized_client_editor_service_accounts" {
  description = "List of service account emails that should be granted Redis editor (read-write) access"
  type        = list(string)
  default     = []
}

variable "secret_accessor_sa_email" {
  description = "The email of the service account that will access the secret."
  type        = string
}

variable "notification_channels" {
  description = "List of notification channels to alert."
  type        = list(string)
}

variable "secret_version_adder" {
  type        = string
  description = "The user allowed to populate new redis auth secret versions."
}

variable "product" {
  description = "Product label to apply to the service."
  type        = string
  default     = "unknown"
}
