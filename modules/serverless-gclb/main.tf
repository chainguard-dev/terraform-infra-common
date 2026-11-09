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

module "domain" {
  source = "../dns-certificates"

  project_id   = var.project_id
  dns_zone     = var.dns_zone
  ipv4_address = google_compute_global_address.this.address
  ipv6_address = one(google_compute_global_address.this-v6[*].address)
  enable_ipv6  = var.enable_ipv6

  service_hostnames = var.public-services
  bucket_hostnames  = var.buckets

  manage_certificates = var.certificate_map == "" || var.retain_managed_certificates
}

module "load_balancer" {
  source = "../serverless-load-balancer"

  project_id      = var.project_id
  name            = var.name
  ipv4_address    = google_compute_global_address.this.id
  ipv6_address    = one(google_compute_global_address.this-v6[*].id)
  enable_ipv6     = var.enable_ipv6
  regions         = var.regions
  serving_regions = var.serving_regions
  public-services = var.public-services
  buckets         = var.buckets

  iap                            = var.iap
  security-policy                = var.security-policy
  forwarding_rule_load_balancing = var.forwarding_rule_load_balancing

  ssl_certificates = module.domain.ssl_certificates
  certificate_map  = var.certificate_map
}
