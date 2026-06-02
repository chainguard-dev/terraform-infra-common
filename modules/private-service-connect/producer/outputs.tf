# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

output "service_attachment_id" {
  description = "Self-link / id of the PSC service attachment. This is the value handed to the consumer module's service_attachment input."
  value       = google_compute_service_attachment.this.id
}

output "internal_lb_ip" {
  description = "Internal VIP of the regional internal ALB frontend."
  value       = google_compute_forwarding_rule.this.ip_address
}
