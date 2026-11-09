# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

# Resource configuration from 24a10a78ff23e4cc758df1037ca0a560ca48ab3a.
# Keep this fixture independent of the extracted modules so upgrades start
# with the resource addresses and configuration used before extraction.
terraform {
  required_version = ">= 1.9"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = ">= 8.7.0"
    }
  }
}

resource "google_compute_global_address" "this" {
  project    = var.project_id
  name       = var.name
  ip_version = "IPV4"
}

resource "google_compute_global_address" "this-v6" {
  count      = var.enable_ipv6 ? 1 : 0
  project    = var.project_id
  name       = "${var.name}-v6"
  ip_version = "IPV6"
}

resource "google_dns_record_set" "public-service" {
  for_each = var.public-services

  project      = var.project_id
  name         = "${each.key}."
  managed_zone = var.dns_zone
  type         = "A"
  ttl          = 60
  rrdatas      = [google_compute_global_address.this.address]
}

resource "google_dns_record_set" "public-service-v6" {
  for_each = var.enable_ipv6 ? var.public-services : {}

  project      = var.project_id
  name         = "${each.key}."
  managed_zone = var.dns_zone
  type         = "AAAA"
  ttl          = 60
  rrdatas      = [google_compute_global_address.this-v6[0].address]
}

locals {
  manage_per_host_certs = var.certificate_map == "" || var.retain_managed_certificates
  buckets               = { for k, v in var.buckets : k => v if !v.disabled }
  regional-backends = merge([
    for svcinfo in values(var.public-services) : merge([
      for region in var.regions : {
        "${svcinfo.name}-${region}" : {
          name     = svcinfo.name
          region   = region
          disabled = svcinfo.disabled
        }
      }
    ]...)
  ]...)
}

resource "google_compute_managed_ssl_certificate" "public-service" {
  for_each = local.manage_per_host_certs ? var.public-services : {}

  name = each.value.name

  managed {
    domains = [google_dns_record_set.public-service[each.key].name]
  }
}

resource "google_dns_record_set" "bucket" {
  for_each = var.buckets

  project      = var.project_id
  name         = "${each.key}."
  managed_zone = var.dns_zone
  type         = "A"
  ttl          = 60
  rrdatas      = [google_compute_global_address.this.address]
}

resource "google_dns_record_set" "bucket-v6" {
  for_each = var.enable_ipv6 ? var.buckets : {}

  project      = var.project_id
  name         = "${each.key}."
  managed_zone = var.dns_zone
  type         = "AAAA"
  ttl          = 60
  rrdatas      = [google_compute_global_address.this-v6[0].address]
}

resource "google_compute_managed_ssl_certificate" "bucket" {
  for_each = local.manage_per_host_certs ? var.buckets : {}

  name = each.value.name

  managed {
    domains = [google_dns_record_set.bucket[each.key].name]
  }
}

data "google_storage_bucket" "buckets" {
  for_each = local.buckets

  name = each.value.bucket_name
}

resource "google_compute_backend_bucket" "buckets" {
  for_each = local.buckets

  project     = var.project_id
  name        = "${var.name}-${each.value.name}"
  bucket_name = data.google_storage_bucket.buckets[each.key].name
  enable_cdn  = each.value.enable_cdn

  dynamic "cdn_policy" {
    for_each = each.value.cdn_policy == null ? [] : [each.value.cdn_policy]
    content {
      cache_mode                   = cdn_policy.value.cache_mode
      client_ttl                   = cdn_policy.value.client_ttl
      default_ttl                  = cdn_policy.value.default_ttl
      max_ttl                      = cdn_policy.value.max_ttl
      signed_url_cache_max_age_sec = cdn_policy.value.signed_url_cache_max_age_sec
    }
  }
}

resource "google_compute_region_network_endpoint_group" "regional-backends" {
  for_each = local.regional-backends

  name                  = each.value.name
  network_endpoint_type = "SERVERLESS"
  region                = each.value.region
  cloud_run {
    service = each.value.name
  }
}

resource "google_compute_backend_service" "public-services" {
  for_each = var.public-services

  project = var.project_id
  name    = each.value.name

  load_balancing_scheme                         = each.value.load_balancing_scheme
  external_managed_migration_state              = each.value.external_managed_migration_state
  external_managed_migration_testing_percentage = each.value.external_managed_migration_testing_percentage
  connection_draining_timeout_sec               = each.value.connection_draining_timeout_sec

  dynamic "backend" {
    for_each = toset(var.serving_regions)
    content {
      group = google_compute_region_network_endpoint_group.regional-backends["${each.value.name}-${backend.key}"]["id"]
    }
  }

  dynamic "iap" {
    for_each = var.iap[*]
    content {
      oauth2_client_id     = iap.value["oauth2_client_id"]
      oauth2_client_secret = iap.value["oauth2_client_secret"]
      enabled              = iap.value["enabled"]
    }
  }

  security_policy = var.security-policy
}

resource "google_compute_url_map" "public-service" {
  project = var.project_id
  name    = var.name

  default_url_redirect {
    host_redirect = "chainguard.dev"
    strip_query   = true
  }

  dynamic "host_rule" {
    for_each = { for k, v in var.public-services : k => v if !v.disabled }
    content {
      hosts        = [host_rule.key]
      path_matcher = host_rule.value.name
    }
  }

  dynamic "path_matcher" {
    for_each = { for k, v in var.public-services : k => v if !v.disabled }
    content {
      name            = path_matcher.value.name
      default_service = google_compute_backend_service.public-services[path_matcher.key].id

      dynamic "default_route_action" {
        for_each = path_matcher.value.cors_policy[*]
        content {
          cors_policy {
            allow_origins        = default_route_action.value.allow_origins
            allow_origin_regexes = default_route_action.value.allow_origin_regexes
            allow_methods        = default_route_action.value.allow_methods
            allow_headers        = default_route_action.value.allow_headers
            expose_headers       = default_route_action.value.expose_headers
            max_age              = default_route_action.value.max_age
            allow_credentials    = default_route_action.value.allow_credentials
          }
        }
      }
    }
  }

  dynamic "host_rule" {
    for_each = local.buckets
    content {
      hosts        = [host_rule.key]
      path_matcher = host_rule.value.name
    }
  }

  dynamic "path_matcher" {
    for_each = local.buckets
    content {
      name            = path_matcher.value.name
      default_service = google_compute_backend_bucket.buckets[path_matcher.key].id
    }
  }
}

resource "google_compute_ssl_policy" "ssl_policy" {
  name                      = "${var.name}-ssl-policy"
  profile                   = "MODERN"
  min_tls_version           = "TLS_1_2"
  post_quantum_key_exchange = "ENABLED"
}

resource "google_compute_target_https_proxy" "public-service" {
  project = var.project_id
  name    = var.name
  url_map = google_compute_url_map.public-service.id

  ssl_certificates = local.manage_per_host_certs ? concat(
    [for domain, cert in google_compute_managed_ssl_certificate.public-service : cert.id if !var.public-services[domain].disabled],
    [for domain, cert in google_compute_managed_ssl_certificate.bucket : cert.id if !var.buckets[domain].disabled],
  ) : null
  certificate_map = var.certificate_map != "" ? var.certificate_map : null
  ssl_policy      = google_compute_ssl_policy.ssl_policy.id
}

resource "google_compute_global_forwarding_rule" "this" {
  project     = var.project_id
  name        = var.name
  ip_protocol = "TCP"
  port_range  = 443
  ip_address  = google_compute_global_address.this.id
  target      = google_compute_target_https_proxy.public-service.id

  external_managed_backend_bucket_migration_state              = var.forwarding_rule_load_balancing.external_managed_backend_bucket_migration_state
  external_managed_backend_bucket_migration_testing_percentage = var.forwarding_rule_load_balancing.external_managed_backend_bucket_migration_testing_percentage
  load_balancing_scheme                                        = var.forwarding_rule_load_balancing.load_balancing_scheme
}

resource "google_compute_global_forwarding_rule" "this-v6" {
  count       = var.enable_ipv6 ? 1 : 0
  project     = var.project_id
  name        = "${var.name}-v6"
  ip_protocol = "TCP"
  port_range  = 443
  ip_address  = google_compute_global_address.this-v6[0].id
  target      = google_compute_target_https_proxy.public-service.id

  external_managed_backend_bucket_migration_state              = var.forwarding_rule_load_balancing.external_managed_backend_bucket_migration_state
  external_managed_backend_bucket_migration_testing_percentage = var.forwarding_rule_load_balancing.external_managed_backend_bucket_migration_testing_percentage
  load_balancing_scheme                                        = var.forwarding_rule_load_balancing.load_balancing_scheme
}

output "resource_ids" {
  value = concat(
    [google_compute_global_address.this.id],
    google_compute_global_address.this-v6[*].id,
    [for record in google_dns_record_set.public-service : record.id],
    [for record in google_dns_record_set.public-service-v6 : record.id],
    [for record in google_dns_record_set.bucket : record.id],
    [for record in google_dns_record_set.bucket-v6 : record.id],
    [for cert in google_compute_managed_ssl_certificate.public-service : cert.id],
    [for cert in google_compute_managed_ssl_certificate.bucket : cert.id],
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
