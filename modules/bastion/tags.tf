// Copyright 2026 Chainguard, Inc.
// SPDX-License-Identifier: Apache-2.0

data "google_project" "resource_manager_tags" {
  count = length(var.resource_manager_tags) > 0 ? 1 : 0

  project_id = var.project_id
}

resource "google_tags_location_tag_binding" "this" {
  for_each = var.resource_manager_tags

  parent    = "//compute.googleapis.com/projects/${data.google_project.resource_manager_tags[0].number}/zones/${var.zone}/instances/${google_compute_instance.bastion.instance_id}"
  tag_value = each.value
  location  = var.zone
}
