// Copyright 2026 Chainguard, Inc.
// SPDX-License-Identifier: Apache-2.0

// The binding parent needs the numeric project number. Read only when tags are set.
data "google_project" "resource_manager_tags" {
  count = length(var.resource_manager_tags) > 0 ? 1 : 0

  project_id = var.project_id
}

resource "google_tags_location_tag_binding" "valkey" {
  for_each = var.resource_manager_tags

  parent    = "//memorystore.googleapis.com/projects/${data.google_project.resource_manager_tags[0].number}/locations/${var.region}/instances/${google_memorystore_instance.valkey.instance_id}"
  tag_value = each.value
  location  = var.region
}
