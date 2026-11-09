# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

output "backend_services" {
  description = "Module-managed service backends by hostname, for backend-scoped IAM and consumer tests. Excludes inherited and out-of-band access."
  value = {
    for host, backend in google_compute_backend_service.public-services : host => {
      name                  = backend.name
      id                    = backend.id
      iap_enabled           = one(backend.iap[*].enabled)
      enable_cdn            = backend.enable_cdn
      load_balancing_scheme = backend.load_balancing_scheme
    }
  }
}

output "backend_buckets" {
  description = "Module-managed backend buckets by hostname. Callers can attach signed-URL keys using the backend name."
  value = {
    for host, backend in google_compute_backend_bucket.buckets : host => {
      name = backend.name
      id   = backend.id
    }
  }
}

output "resource_ids" {
  description = "IDs of all NEGs, backends, URL map, TLS policy, HTTPS proxy and forwarding rules managed by this module."
  value = concat(
    [for backend in google_compute_backend_bucket.buckets : backend.id],
    [for backend in google_compute_backend_service.public-services : backend.id],
    [for neg in google_compute_region_network_endpoint_group.regional-backends : neg.id],
    [
      google_compute_url_map.public-service.id,
      google_compute_ssl_policy.ssl_policy.id,
      google_compute_target_https_proxy.public-service.id,
      google_compute_global_forwarding_rule.this.id,
    ],
    google_compute_global_forwarding_rule.this-v6[*].id,
  )
}
