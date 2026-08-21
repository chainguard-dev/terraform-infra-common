// Copyright 2026 Chainguard, Inc.
// SPDX-License-Identifier: Apache-2.0

resource "google_tags_location_tag_binding" "this" {
  for_each = var.resource_manager_tags

  parent    = "//storage.googleapis.com/projects/_/buckets/${google_storage_bucket.status.name}"
  tag_value = each.value
  location  = lower(google_storage_bucket.status.location)
}
