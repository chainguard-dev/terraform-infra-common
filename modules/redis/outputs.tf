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

output "id" {
  description = "Redis instance ID."
  value       = google_redis_instance.default.id
}

output "host" {
  description = "The IP address of the instance."
  value       = google_redis_instance.default.host
}

output "port" {
  description = "The port number of the instance."
  value       = google_redis_instance.default.port
}

output "region" {
  description = "The region the instance lives in."
  value       = google_redis_instance.default.region
}

output "current_location_id" {
  description = "The zone where the instance is currently located."
  value       = google_redis_instance.default.current_location_id
}

output "redis_version" {
  description = "The version of Redis software."
  value       = google_redis_instance.default.redis_version
}

output "memory_size_gb" {
  description = "Redis memory size in GiB."
  value       = google_redis_instance.default.memory_size_gb
}

output "connection_name" {
  description = "The connection name of the instance to be used in connection strings."
  value       = "${var.project_id}:${google_redis_instance.default.region}:${google_redis_instance.default.name}"
}

output "uri" {
  description = "The connection URI to be used for accessing Redis."
  value       = "redis://${google_redis_instance.default.host}:${google_redis_instance.default.port}"
  depends_on  = [google_redis_instance.default]
}

output "persistence_mode" {
  description = "The persistence mode of the Redis instance."
  value       = var.persistence_config.persistence_mode
}

output "rdb_snapshot_period" {
  description = "The snapshot period for RDB persistence."
  value       = var.persistence_config.persistence_mode == "RDB" ? var.persistence_config.rdb_snapshot_period : null
}

output "auth_secret_id" {
  description = "The ID of the Secret Manager secret containing the Redis AUTH string"
  value       = var.auth_enabled ? module.redis_auth_secret[0].secret_id : null
}

output "auth_enabled" {
  description = "Whether AUTH is enabled for the Redis instance"
  value       = var.auth_enabled
}
