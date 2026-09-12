// Copyright 2026 Chainguard, Inc.
// SPDX-License-Identifier: Apache-2.0

terraform {
  required_providers {
    google      = { source = "hashicorp/google" }
    google-beta = { source = "hashicorp/google-beta" }
  }
}

locals {
  iap_enabled = length(var.iap_members) > 0
}

// The API is shared by services in this project. Removing one service's
// opt-in must not disable IAP for the others.
resource "google_project_service" "iap" {
  count = local.iap_enabled ? 1 : 0

  project            = var.project_id
  service            = "iap.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service_identity" "iap" {
  provider = google-beta
  count    = local.iap_enabled ? 1 : 0

  project = var.project_id
  service = google_project_service.iap[0].service
}

// Cloud Run sees IAP's identity after IAP authenticates the browser user.
resource "google_cloud_run_v2_service_iam_member" "iap-invoker" {
  for_each = local.iap_enabled ? var.regions : {}

  project  = var.project_id
  location = each.key
  name     = google_cloud_run_v2_service.this[each.key].name
  role     = "roles/run.invoker"
  member   = "serviceAccount:${google_project_service_identity.iap[0].email}"
}

// Own this role's service-level membership so removing a member revokes its
// direct grant. This does not override access inherited from parent policies.
resource "google_iap_web_cloud_run_service_iam_binding" "access" {
  for_each = local.iap_enabled ? var.regions : {}

  project                = var.project_id
  location               = each.key
  cloud_run_service_name = google_cloud_run_v2_service.this[each.key].name
  role                   = "roles/iap.httpsResourceAccessor"
  members                = var.iap_members
}
