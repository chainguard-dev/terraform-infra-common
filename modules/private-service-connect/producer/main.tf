# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

terraform {
  required_providers {
    google = {
      source = "hashicorp/google"
    }
  }
}

# Serverless NEG fronting the existing regional Cloud Run service.
resource "google_compute_region_network_endpoint_group" "this" {
  project               = var.project
  region                = var.region
  name                  = "${var.name}-neg"
  network_endpoint_type = "SERVERLESS"

  cloud_run {
    service = var.cloud_run_service_name
  }
}

# Regional internal-managed backend service pointing at the serverless NEG.
# Serverless NEG backends do not use health checks.
resource "google_compute_region_backend_service" "this" {
  project               = var.project
  region                = var.region
  name                  = "${var.name}-backend"
  load_balancing_scheme = "INTERNAL_MANAGED"
  protocol              = "HTTP"

  backend {
    group = google_compute_region_network_endpoint_group.this.id
  }
}

# Regional URL map routing all traffic to the backend service.
resource "google_compute_region_url_map" "this" {
  project         = var.project
  region          = var.region
  name            = "${var.name}-urlmap"
  default_service = google_compute_region_backend_service.this.id
}

# Regional target HTTP proxy. HTTP is intentional: TLS to the run.app
# backend is handled by the serverless NEG, and inbound authorization is
# enforced via Cloud Run invoker IAM (added in a later phase, not here).
resource "google_compute_region_target_http_proxy" "this" {
  project = var.project
  region  = var.region
  name    = "${var.name}-http-proxy"
  url_map = google_compute_region_url_map.this.id
}

# Internal ALB frontend. The REGIONAL_MANAGED_PROXY subnet must exist in the
# region before this forwarding rule can be created; the caller passes its
# self-link so we can express that ordering explicitly.
resource "google_compute_forwarding_rule" "this" {
  project               = var.project
  region                = var.region
  name                  = "${var.name}-fr"
  load_balancing_scheme = "INTERNAL_MANAGED"
  network               = var.network
  subnetwork            = var.subnetwork
  target                = google_compute_region_target_http_proxy.this.id
  ports                 = ["80"]

  labels = var.labels

  depends_on = [var.proxy_only_subnet]
}

# PSC service attachment publishing the internal ALB to accepted consumers.
resource "google_compute_service_attachment" "this" {
  project               = var.project
  region                = var.region
  name                  = "${var.name}-sa"
  target_service        = google_compute_forwarding_rule.this.id
  connection_preference = "ACCEPT_MANUAL"
  nat_subnets           = var.psc_nat_subnets
  enable_proxy_protocol = false

  dynamic "consumer_accept_lists" {
    for_each = var.consumer_accept_projects
    content {
      project_id_or_num = consumer_accept_lists.value
      connection_limit  = var.connection_limit
    }
  }
}
