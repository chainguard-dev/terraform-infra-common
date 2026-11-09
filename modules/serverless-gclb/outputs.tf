/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

output "frontend" {
  description = "Global frontend IP addresses. ipv6_address is null when IPv6 is disabled."
  value = {
    ipv4_address = google_compute_global_address.this.address
    ipv6_address = one(google_compute_global_address.this-v6[*].address)
  }
}

output "backend_services" {
  description = "Module-managed backend services keyed by hostname, with identifiers for backend-scoped IAM and security configuration for consumer tests. Excludes inherited and out-of-band access."
  value       = module.load_balancer.backend_services
}

output "backend_buckets" {
  description = "The backend buckets created for buckets, keyed by hostname: name and id. A caller serving a bucket to signed URLs only attaches its Cloud CDN signed-URL keys to the name, in Terraform or out of band so the key values never enter state, and grants Cloud CDN's fill service agent read on the storage bucket; the id is what the URL map routes the hostname's traffic through."
  value       = module.load_balancer.backend_buckets
}

output "resource_ids" {
  description = "IDs of all frontend addresses, DNS records, certificates and load-balancing resources managed by this assembly."
  value = concat(
    [google_compute_global_address.this.id],
    google_compute_global_address.this-v6[*].id,
    module.domain.resource_ids,
    module.load_balancer.resource_ids,
  )
}
