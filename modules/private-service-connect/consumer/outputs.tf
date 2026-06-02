# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

output "endpoint_ip" {
  description = "Internal IP address assigned to the PSC endpoint."
  value       = local.endpoint_ip
}

output "psc_connection_id" {
  description = "The PSC connection id of the endpoint forwarding rule."
  value       = google_compute_forwarding_rule.this.psc_connection_id
}
