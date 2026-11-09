# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

output "configuration" {
  sensitive = true
  value = {
    backend_services  = google_compute_backend_service.public-services
    backend_buckets   = google_compute_backend_bucket.buckets
    regional_backends = google_compute_region_network_endpoint_group.regional-backends
    url_map           = google_compute_url_map.public-service
    ssl_policy        = google_compute_ssl_policy.ssl_policy
    https_proxy       = google_compute_target_https_proxy.public-service
    forwarding_rules = {
      ipv4 = google_compute_global_forwarding_rule.this
      ipv6 = google_compute_global_forwarding_rule.this-v6
    }
  }
}
