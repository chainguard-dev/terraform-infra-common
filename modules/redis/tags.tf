// Copyright 2026 Chainguard, Inc.
// SPDX-License-Identifier: Apache-2.0

data "google_project" "resource_manager_tags" {
  count = length(var.resource_manager_tags) > 0 ? 1 : 0

  project_id = var.project_id
}

resource "google_tags_location_tag_binding" "this" {
  for_each = var.resource_manager_tags

  parent    = "//redis.googleapis.com/projects/${data.google_project.resource_manager_tags[0].number}/locations/${var.region}/instances/${google_redis_instance.default.name}"
  tag_value = each.value
  location  = var.region
}
