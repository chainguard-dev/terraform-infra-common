// Copyright 2026 Chainguard, Inc.
// SPDX-License-Identifier: Apache-2.0

locals {
  project_number = var.project_number != null ? var.project_number : one(data.google_project.resource_manager_tags[*].number)

  resource_manager_tag_bindings = {
    for pair in setproduct(keys(var.regions), keys(var.resource_manager_tags)) :
    "${pair[0]}/${pair[1]}" => {
      location  = pair[0]
      tag_value = var.resource_manager_tags[pair[1]]
    }
  }
}

data "google_project" "resource_manager_tags" {
  count = length(var.resource_manager_tags) > 0 && var.project_number == null ? 1 : 0

  project_id = var.project_id
}

resource "google_tags_location_tag_binding" "this" {
  for_each = local.resource_manager_tag_bindings

  parent    = "//run.googleapis.com/projects/${local.project_number}/locations/${each.value.location}/jobs/${google_cloud_run_v2_job.this[each.value.location].name}"
  tag_value = each.value.tag_value
  location  = each.value.location
}
