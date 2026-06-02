# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

terraform {
  required_providers {
    google = {
      source = "hashicorp/google"
    }
  }
}

locals {
  # Use the caller-supplied address when provided, otherwise the one we reserve.
  reserve_address = var.address == ""
  endpoint_address = local.reserve_address ? (
    google_compute_address.this[0].id
  ) : var.address

  # Literal dotted-quad IP for the endpoint_ip output. The forwarding rule is
  # configured from the reserved address's self-link (endpoint_address), so
  # its ip_address attribute can read back as a self-link rather than an IP;
  # surface the address resource's literal IP instead (the proven Cloud SQL
  # PSC consumer pattern). For a caller-supplied address we echo it back.
  endpoint_ip = local.reserve_address ? google_compute_address.this[0].address : var.address
}

# Reserve an internal IP for the PSC endpoint when the caller did not supply one.
resource "google_compute_address" "this" {
  count = local.reserve_address ? 1 : 0

  project      = var.project
  region       = var.region
  name         = "${var.name}-ip"
  address_type = "INTERNAL"
  subnetwork   = var.subnetwork
}

# The PSC endpoint: a forwarding rule targeting the producer's service
# attachment. load_balancing_scheme is empty for PSC endpoints.
resource "google_compute_forwarding_rule" "this" {
  project                 = var.project
  region                  = var.region
  name                    = "${var.name}-endpoint"
  load_balancing_scheme   = "" # Empty for PSC endpoints.
  network                 = var.network
  subnetwork              = var.subnetwork
  ip_address              = local.endpoint_address
  target                  = var.service_attachment
  allow_psc_global_access = false
}
