# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

terraform {
  required_version = ">= 1.9"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = ">= 8.7.0"
    }
  }
}

provider "google" {
  alias   = "dns"
  project = "dns-project"
}

module "domain" {
  source = "../../../dns-certificates"

  providers = {
    google = google.dns
  }

  project_id             = "dns-project"
  certificate_project_id = "workload-project"
  dns_zone               = "existing-zone"
  ipv4_address           = "192.0.2.10"
  service_hostnames = {
    "receiver.example.dev" = {
      name = "receiver"
    }
  }
}

module "load_balancer" {
  source = "../../../serverless-load-balancer"

  project_id       = "workload-project"
  name             = "receiver"
  ipv4_address     = "projects/workload-project/global/addresses/receiver"
  ssl_certificates = module.domain.ssl_certificates
  public-services = {
    "receiver.example.dev" = {
      name                  = "receiver"
      load_balancing_scheme = "EXTERNAL_MANAGED"
    }
  }
  forwarding_rule_load_balancing = {
    load_balancing_scheme = "EXTERNAL_MANAGED"
  }
  iap = {
    enabled = true
  }
}

output "backend_services" {
  value = module.load_balancer.backend_services
}

output "records" {
  value = module.domain.records
}

output "domain_resource_ids" {
  value = module.domain.resource_ids
}

output "ssl_certificates" {
  value = module.domain.ssl_certificates
}
