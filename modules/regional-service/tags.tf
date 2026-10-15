// Copyright 2026 Chainguard, Inc.
// SPDX-License-Identifier: Apache-2.0

locals {
  project_number = var.project_number != null ? var.project_number : data.google_project.project[0].number

  resource_manager_tag_bindings = {
    for pair in setproduct(keys(var.regions), keys(var.resource_manager_tags)) :
    "${pair[0]}/${pair[1]}" => {
      location  = pair[0]
      tag_value = var.resource_manager_tags[pair[1]]
    }
  }
}

resource "google_tags_location_tag_binding" "this" {
  for_each = local.resource_manager_tag_bindings

  parent    = "//run.googleapis.com/projects/${local.project_number}/locations/${each.value.location}/services/${google_cloud_run_v2_service.this[each.value.location].name}"
  tag_value = each.value.tag_value
  location  = each.value.location
}
