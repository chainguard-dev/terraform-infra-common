# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

mock_provider "google" {
  mock_resource "google_compute_backend_service" {
    defaults = {
      enable_cdn = false
    }
  }
}

variables {
  project_id  = "fixture-project"
  name        = "fixture"
  dns_zone    = "fixture-zone"
  team        = "fixture"
  enable_ipv6 = true
  regions = [
    "us-central1",
    "us-east1",
  ]
  serving_regions = ["us-east1"]
  public-services = {
    "api.example.dev" = {
      name                  = "api"
      load_balancing_scheme = "EXTERNAL_MANAGED"
      cors_policy = {
        allow_origins = ["https://console.example.dev"]
      }
    }
    "disabled.example.dev" = {
      name                  = "disabled"
      disabled              = true
      load_balancing_scheme = "EXTERNAL_MANAGED"
    }
  }
  buckets = {
    "downloads.example.dev" = {
      name        = "downloads"
      bucket_name = "fixture-downloads"
      cdn_policy = {
        cache_mode                   = "CACHE_ALL_STATIC"
        client_ttl                   = 60
        default_ttl                  = 60
        max_ttl                      = 3600
        signed_url_cache_max_age_sec = 600
      }
    }
    "disabled-downloads.example.dev" = {
      name        = "disabled-downloads"
      bucket_name = "fixture-disabled-downloads"
      disabled    = true
    }
  }
  iap = {
    enabled = true
  }
  security-policy = "projects/fixture-project/global/securityPolicies/fixture"
  forwarding_rule_load_balancing = {
    load_balancing_scheme = "EXTERNAL_MANAGED"
  }

  retained_configuration = {
    certificate_map = "//certificatemanager.googleapis.com/projects/fixture-project/locations/global/certificateMaps/fixture"
    public_services = {
      "api.example.dev" = {
        name                                          = "api"
        load_balancing_scheme                         = "EXTERNAL"
        external_managed_migration_state              = "TEST_BY_PERCENTAGE"
        external_managed_migration_testing_percentage = 25
        connection_draining_timeout_sec               = 60
      }
    }
    forwarding_rule_load_balancing = {
      load_balancing_scheme                                        = "EXTERNAL_MANAGED"
      external_managed_backend_bucket_migration_state              = "TEST_BY_PERCENTAGE"
      external_managed_backend_bucket_migration_testing_percentage = 25
    }
  }
}

run "legacy_dualstack" {
  state_key = "dualstack"

  module {
    source = "./tests/legacy"
  }
}

run "upgrade_dualstack" {
  command   = plan
  state_key = "dualstack"

  module {
    source = "./tests/upgrade/assembly"
  }

  assert {
    condition     = output.resource_ids == run.legacy_dualstack.resource_ids
    error_message = "Extraction must preserve every managed resource ID, including disabled hostnames, buckets and regional NEGs."
  }

  assert {
    condition = jsonencode(output.configuration) == jsonencode(merge(run.legacy_dualstack.configuration, {
      backend_services = {
        for host, backend in run.legacy_dualstack.configuration.backend_services : host => merge(backend, {
          enable_cdn = false
        })
      }
    }))
    error_message = "Extraction must preserve the complete dual-stack load-balancer configuration except for explicitly disabling service-backend CDN."
  }

  assert {
    condition     = jsonencode(output.records) == jsonencode(run.legacy_dualstack.records)
    error_message = "Extraction must preserve every dual-stack DNS record's project, zone, name, type, TTL and address values, including disabled hostnames."
  }
}

run "legacy_ipv4_certificate_map" {
  state_key = "ipv4-map"

  module {
    source = "./tests/legacy"
  }

  variables {
    enable_ipv6     = false
    certificate_map = "//certificatemanager.googleapis.com/projects/fixture-project/locations/global/certificateMaps/fixture"
  }
}

run "upgrade_ipv4_certificate_map" {
  command   = plan
  state_key = "ipv4-map"

  module {
    source = "./tests/upgrade/assembly"
  }

  variables {
    enable_ipv6     = false
    certificate_map = "//certificatemanager.googleapis.com/projects/fixture-project/locations/global/certificateMaps/fixture"
  }

  assert {
    condition     = output.resource_ids == run.legacy_ipv4_certificate_map.resource_ids
    error_message = "Extraction must preserve IPv4-only deployments that use a certificate map."
  }

  assert {
    condition = jsonencode(output.configuration) == jsonencode(merge(run.legacy_ipv4_certificate_map.configuration, {
      backend_services = {
        for host, backend in run.legacy_ipv4_certificate_map.configuration.backend_services : host => merge(backend, {
          enable_cdn = false
        })
      }
    }))
    error_message = "Extraction must preserve the complete IPv4 certificate-map configuration except for explicitly disabling service-backend CDN."
  }

  assert {
    condition     = jsonencode(output.records) == jsonencode(run.legacy_ipv4_certificate_map.records)
    error_message = "Extraction must preserve every IPv4-only DNS record's project, zone, name, type, TTL and address values."
  }
}

run "legacy_retained_certificates" {
  state_key = "retained-certificates"

  module {
    source = "./tests/legacy"
  }

  variables {
    certificate_map                = var.retained_configuration.certificate_map
    retain_managed_certificates    = true
    public-services                = var.retained_configuration.public_services
    forwarding_rule_load_balancing = var.retained_configuration.forwarding_rule_load_balancing
  }
}

run "upgrade_retained_certificates" {
  command   = plan
  state_key = "retained-certificates"

  module {
    source = "./tests/upgrade/assembly"
  }

  variables {
    certificate_map                = var.retained_configuration.certificate_map
    retain_managed_certificates    = true
    public-services                = var.retained_configuration.public_services
    forwarding_rule_load_balancing = var.retained_configuration.forwarding_rule_load_balancing
  }

  assert {
    condition     = output.resource_ids == run.legacy_retained_certificates.resource_ids
    error_message = "Extraction must preserve retained certificates and load-balancer migration settings."
  }

  assert {
    condition = jsonencode(output.configuration) == jsonencode(merge(run.legacy_retained_certificates.configuration, {
      backend_services = {
        for host, backend in run.legacy_retained_certificates.configuration.backend_services : host => merge(backend, {
          enable_cdn = false
        })
      }
    }))
    error_message = "Extraction must preserve the complete retained-certificate migration configuration except for explicitly disabling service-backend CDN."
  }

  assert {
    condition     = jsonencode(output.records) == jsonencode(run.legacy_retained_certificates.records)
    error_message = "Extraction must preserve every DNS record during a retained-certificate migration."
  }
}
