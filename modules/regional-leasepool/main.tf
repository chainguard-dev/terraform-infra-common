# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

terraform {
  required_providers {
    google = { source = "hashicorp/google" }
    random = { source = "hashicorp/random" }
  }
}

resource "random_id" "suffix" {
  byte_length = 4
}

resource "google_storage_bucket" "leases" {
  for_each = var.regions

  project       = var.project_id
  name          = "${var.name}-${each.key}-${random_id.suffix.hex}"
  location      = each.key
  storage_class = "STANDARD"
  force_destroy = var.force_destroy

  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"

  # Lease renewal overwrites small objects frequently. Historical versions are
  # not a recovery mechanism: restoring old ownership would violate the protocol.
  versioning {
    enabled = false
  }
  soft_delete_policy {
    retention_duration_seconds = 0
  }

  labels = merge({
    terraform-module = "regional-leasepool"
    name             = var.name
    squad            = var.team
  }, var.labels)
}

resource "google_storage_bucket_iam_member" "participant" {
  for_each = var.regions

  bucket = google_storage_bucket.leases[each.key].name
  role   = "roles/storage.objectUser"
  member = var.identity
}
