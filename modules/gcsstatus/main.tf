/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

# GCS bucket for storing reconciliation status as JSON objects, consumed by the
# gcsstatusmanager (public/go-driftlessaf/reconcilers/gcsstatusmanager). Objects
# live at "<identity>/<key>"; the identity prefix is chosen by the reconciler, not
# this module.
resource "random_string" "suffix" {
  length  = 6
  special = false
  upper   = false
  numeric = true
}

resource "google_storage_bucket" "status" {
  project                     = var.project_id
  name                        = "${var.name}-${random_string.suffix.result}"
  location                    = var.location
  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"
  storage_class               = "STANDARD"

  # Status objects are overwritten in place; keeping old versions would only
  # accumulate cost.
  versioning {
    enabled = false
  }

  dynamic "lifecycle_rule" {
    for_each = var.lifecycle_age_days > 0 ? [1] : []
    content {
      action {
        type = "Delete"
      }
      condition {
        age = var.lifecycle_age_days
      }
    }
  }
}

# Writers get objectUser (create/read/overwrite/delete on objects). Unlike the
# ocistatus module — which needs repoAdmin because replacing an OCI attestation
# deletes referrer manifests — a GCS overwrite is a plain write, so no admin/ACL
# privilege is required.
resource "google_storage_bucket_iam_member" "writers" {
  for_each = toset(var.writer_service_accounts)

  bucket = google_storage_bucket.status.name
  role   = "roles/storage.objectUser"
  member = each.value
}

resource "google_storage_bucket_iam_member" "readers" {
  for_each = toset(var.reader_service_accounts)

  bucket = google_storage_bucket.status.name
  role   = "roles/storage.objectViewer"
  member = each.value
}
