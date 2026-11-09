# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

mock_provider "google" {
  mock_resource "google_compute_target_https_proxy" {
    defaults = {
      id = "projects/fixture-project/global/targetHttpsProxies/fixture"
    }
  }
}

variables {
  project_id       = "fixture-project"
  name             = "fixture"
  ipv4_address     = "projects/fixture-project/global/addresses/fixture"
  ipv6_address     = "projects/fixture-project/global/addresses/fixture-v6"
  ssl_certificates = ["projects/fixture-project/global/sslCertificates/api"]
  certificate_map  = "//certificatemanager.googleapis.com/projects/fixture-project/locations/global/certificateMaps/fixture"
  public-services = {
    "api.example.dev" = {
      name = "api"
    }
    "disabled.example.dev" = {
      name     = "disabled"
      disabled = true
    }
  }
  buckets = {
    "downloads.example.dev" = {
      name        = "downloads"
      bucket_name = "fixture-downloads"
    }
    "disabled-downloads.example.dev" = {
      name        = "disabled-downloads"
      bucket_name = "fixture-disabled-downloads"
      disabled    = true
    }
  }
}

run "supplied_frontend_and_certificate_references" {
  command = apply

  variables {
    enable_ipv6 = true
  }

  assert {
    condition = jsonencode({
      certificates = google_compute_target_https_proxy.public-service.ssl_certificates
      map          = google_compute_target_https_proxy.public-service.certificate_map
      rules = [for rule in concat(
        [google_compute_global_forwarding_rule.this],
        google_compute_global_forwarding_rule.this-v6,
        ) : {
        name    = rule.name
        address = rule.ip_address
        port    = rule.port_range
        target  = rule.target
      }]
      }) == jsonencode({
      certificates = ["projects/fixture-project/global/sslCertificates/api"]
      map          = "//certificatemanager.googleapis.com/projects/fixture-project/locations/global/certificateMaps/fixture"
      rules = [
        {
          name    = "fixture"
          address = "projects/fixture-project/global/addresses/fixture"
          port    = "443"
          target  = "projects/fixture-project/global/targetHttpsProxies/fixture"
        },
        {
          name    = "fixture-v6"
          address = "projects/fixture-project/global/addresses/fixture-v6"
          port    = "443"
          target  = "projects/fixture-project/global/targetHttpsProxies/fixture"
        },
      ]
    })
    error_message = "The frontend must use the supplied addresses and both certificate sources during a map migration."
  }

  assert {
    condition = jsonencode({
      hosts            = google_compute_url_map.public-service.host_rule
      service_backends = keys(google_compute_backend_service.public-services)
      bucket_backends  = keys(google_compute_backend_bucket.buckets)
      }) == jsonencode({
      hosts = [
        {
          description  = null
          hosts        = ["api.example.dev"]
          path_matcher = "api"
        },
        {
          description  = null
          hosts        = ["downloads.example.dev"]
          path_matcher = "downloads"
        },
      ]
      service_backends = ["api.example.dev", "disabled.example.dev"]
      bucket_backends  = ["downloads.example.dev"]
    })
    error_message = "Disabled hosts must have no routes, while disabled services retain their backends."
  }
}

run "ipv4_only_omits_ipv6_forwarding_rule" {
  command = plan

  assert {
    condition     = google_compute_global_forwarding_rule.this-v6 == []
    error_message = "IPv4-only configurations must omit the IPv6 forwarding rule."
  }
}

run "ipv6_requires_an_address" {
  command = plan

  variables {
    enable_ipv6  = true
    ipv6_address = null
  }

  expect_failures = [var.ipv6_address]
}
