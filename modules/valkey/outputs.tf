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

output "id" {
  description = "Valkey instance ID."
  value       = google_memorystore_instance.valkey.id
}

output "host" {
  description = "The IP address of the instance's PSC connect endpoint (the primary endpoint in CLUSTER_DISABLED mode, the discovery endpoint in CLUSTER mode)."
  value       = local.connection.ip_address
}

output "port" {
  description = "The port of the instance's PSC connect endpoint."
  value       = local.connection.port
}

output "addr" {
  description = "host:port of the instance's PSC connect endpoint."
  value       = "${local.connection.ip_address}:${local.connection.port}"
}

output "reader_addr" {
  description = "host:port of the reader endpoint, for read-scaling standalone clients. Null unless mode is CLUSTER_DISABLED with replicas."
  value       = local.reader != null ? "${local.reader.ip_address}:${local.reader.port}" : null
}

output "ca_pem" {
  description = "The managed server CA bundle (all currently-active CAs, concatenated PEM) clients pin to verify the instance's TLS connection. The CA is stable for years; only the leaf server cert rotates under it, so an apply-time capture stays valid across that rotation. The bundle is public certificate material."
  value       = join("\n", flatten(google_memorystore_instance.valkey.managed_server_ca[0].ca_certs[*].certificates))
}
