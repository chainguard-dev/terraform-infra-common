# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

moved {
  from = google_dns_record_set.public-service
  to   = module.domain.google_dns_record_set.public-service
}

moved {
  from = google_dns_record_set.public-service-v6
  to   = module.domain.google_dns_record_set.public-service-v6
}

moved {
  from = google_dns_record_set.bucket
  to   = module.domain.google_dns_record_set.bucket
}

moved {
  from = google_dns_record_set.bucket-v6
  to   = module.domain.google_dns_record_set.bucket-v6
}

moved {
  from = google_compute_managed_ssl_certificate.public-service
  to   = module.domain.google_compute_managed_ssl_certificate.public-service
}

moved {
  from = google_compute_managed_ssl_certificate.bucket
  to   = module.domain.google_compute_managed_ssl_certificate.bucket
}

moved {
  from = google_compute_backend_bucket.buckets
  to   = module.load_balancer.google_compute_backend_bucket.buckets
}

moved {
  from = google_compute_backend_service.public-services
  to   = module.load_balancer.google_compute_backend_service.public-services
}

moved {
  from = google_compute_region_network_endpoint_group.regional-backends
  to   = module.load_balancer.google_compute_region_network_endpoint_group.regional-backends
}

moved {
  from = google_compute_url_map.public-service
  to   = module.load_balancer.google_compute_url_map.public-service
}

moved {
  from = google_compute_ssl_policy.ssl_policy
  to   = module.load_balancer.google_compute_ssl_policy.ssl_policy
}

moved {
  from = google_compute_target_https_proxy.public-service
  to   = module.load_balancer.google_compute_target_https_proxy.public-service
}

moved {
  from = google_compute_global_forwarding_rule.this
  to   = module.load_balancer.google_compute_global_forwarding_rule.this
}

moved {
  from = google_compute_global_forwarding_rule.this-v6
  to   = module.load_balancer.google_compute_global_forwarding_rule.this-v6
}
