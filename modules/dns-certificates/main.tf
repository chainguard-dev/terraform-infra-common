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

resource "google_dns_record_set" "public-service" {
  for_each = var.service_hostnames

  project      = var.project_id
  name         = "${each.key}."
  managed_zone = var.dns_zone
  type         = "A"
  ttl          = 60
  rrdatas      = [var.ipv4_address]
}

resource "google_dns_record_set" "public-service-v6" {
  for_each = var.enable_ipv6 ? var.service_hostnames : {}

  project      = var.project_id
  name         = "${each.key}."
  managed_zone = var.dns_zone
  type         = "AAAA"
  ttl          = 60
  rrdatas      = [var.ipv6_address]
}

resource "google_compute_managed_ssl_certificate" "public-service" {
  for_each = var.manage_certificates ? var.service_hostnames : {}

  project = var.certificate_project_id
  name    = each.value.name

  managed {
    domains = [google_dns_record_set.public-service[each.key].name]
  }
}

resource "google_dns_record_set" "bucket" {
  for_each = var.bucket_hostnames

  project      = var.project_id
  name         = "${each.key}."
  managed_zone = var.dns_zone
  type         = "A"
  ttl          = 60
  rrdatas      = [var.ipv4_address]
}

resource "google_dns_record_set" "bucket-v6" {
  for_each = var.enable_ipv6 ? var.bucket_hostnames : {}

  project      = var.project_id
  name         = "${each.key}."
  managed_zone = var.dns_zone
  type         = "AAAA"
  ttl          = 60
  rrdatas      = [var.ipv6_address]
}

resource "google_compute_managed_ssl_certificate" "bucket" {
  for_each = var.manage_certificates ? var.bucket_hostnames : {}

  project = var.certificate_project_id
  name    = each.value.name

  managed {
    domains = [google_dns_record_set.bucket[each.key].name]
  }
}
