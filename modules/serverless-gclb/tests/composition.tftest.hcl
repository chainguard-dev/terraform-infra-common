# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

mock_provider "google" {
  mock_resource "google_compute_global_address" {
    defaults = {
      id      = "projects/workload-project/global/addresses/legacy"
      address = "192.0.2.10"
    }
  }

  mock_resource "google_compute_backend_bucket" {
    defaults = {
      id = "projects/workload-project/global/backendBuckets/legacy-downloads"
    }
  }

  mock_resource "google_compute_backend_service" {
    defaults = {
      id = "projects/workload-project/global/backendServices/receiver"
    }
  }
}

variables {
  caller_backends = {
    backend_services = {
      "legacy.example.dev" = {
        name                  = "receiver"
        id                    = "projects/workload-project/global/backendServices/receiver"
        iap_enabled           = null
        enable_cdn            = false
        load_balancing_scheme = "EXTERNAL"
      }
    }
    backend_buckets = {
      "downloads.example.dev" = {
        name = "legacy-downloads"
        id   = "projects/workload-project/global/backendBuckets/legacy-downloads"
      }
    }
  }

  frontend_cases = {
    dualstack = {
      enable_ipv6 = true
      frontend = {
        ipv4_address = "192.0.2.10"
        ipv6_address = "2001:db8::10"
      }
    }
    ipv4 = {
      enable_ipv6 = false
      frontend = {
        ipv4_address = "192.0.2.10"
        ipv6_address = null
      }
    }
  }
}

run "existing_caller_inherits_its_default_provider" {
  command = apply

  module {
    source = "./tests/existing-caller"
  }

  variables {
    enable_ipv6 = var.frontend_cases.dualstack.enable_ipv6
  }

  override_resource {
    target = module.endpoint.google_compute_global_address.this-v6[0]
    values = {
      id      = "projects/workload-project/global/addresses/legacy-v6"
      address = "2001:db8::10"
    }
  }

  assert {
    condition = jsonencode({
      frontend         = output.frontend
      backend_services = output.backend_services
      backend_buckets  = output.backend_buckets
      }) == jsonencode(merge(var.caller_backends, {
        frontend = var.frontend_cases.dualstack.frontend
    }))
    error_message = "Existing callers must preserve dual-stack frontend addresses and complete service and bucket outputs without provider mappings."
  }
}

run "ipv4_only_caller_keeps_backend_outputs" {
  command = apply

  module {
    source = "./tests/existing-caller"
  }

  variables {
    enable_ipv6 = var.frontend_cases.ipv4.enable_ipv6
  }

  assert {
    condition = jsonencode({
      frontend         = output.frontend
      backend_services = output.backend_services
      backend_buckets  = output.backend_buckets
      }) == jsonencode(merge(var.caller_backends, {
        frontend = var.frontend_cases.ipv4.frontend
    }))
    error_message = "IPv4-only callers must export a null IPv6 address without changing the complete service and bucket outputs."
  }
}

mock_provider "google" {
  alias = "dns"

  mock_resource "google_compute_managed_ssl_certificate" {
    defaults = {
      id = "projects/workload-project/global/sslCertificates/receiver"
    }
  }

  mock_resource "google_dns_record_set" {
    defaults = {
      id = "projects/dns-project/managedZones/existing-zone/rrsets/receiver.example.dev./A"
    }
  }
}

run "independent_components_accept_separate_dns_authority" {
  command = apply

  providers = {
    google     = google
    google.dns = google.dns
  }

  module {
    source = "./tests/composition"
  }

  assert {
    condition = jsonencode({
      records             = output.records
      backends            = output.backend_services
      certificates        = output.ssl_certificates
      domain_resource_ids = output.domain_resource_ids
      }) == jsonencode({
      records = [
        {
          project = "dns-project"
          zone    = "existing-zone"
          name    = "receiver.example.dev."
          type    = "A"
          ttl     = 60
          rrdatas = ["192.0.2.10"]
        },
      ]
      backends = {
        "receiver.example.dev" = {
          name                  = "receiver"
          id                    = "projects/workload-project/global/backendServices/receiver"
          iap_enabled           = true
          enable_cdn            = false
          load_balancing_scheme = "EXTERNAL_MANAGED"
        }
      }
      certificates = ["projects/workload-project/global/sslCertificates/receiver"]
      domain_resource_ids = [
        "projects/dns-project/managedZones/existing-zone/rrsets/receiver.example.dev./A",
        "projects/workload-project/global/sslCertificates/receiver",
      ]
    })
    error_message = "The components must compose with independent DNS credentials and a backend-scoped IAP configuration."
  }
}
