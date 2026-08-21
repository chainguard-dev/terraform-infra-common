// Copyright 2026 Chainguard, Inc.
// SPDX-License-Identifier: Apache-2.0

resource "google_tags_location_tag_binding" "cluster" {
  for_each = var.resource_manager_tags

  parent    = "//container.googleapis.com/${google_container_cluster.this.id}"
  tag_value = each.value
  location  = google_container_cluster.this.location
}
