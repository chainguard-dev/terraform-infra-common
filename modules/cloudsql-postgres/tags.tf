// Copyright 2026 Chainguard, Inc.
// SPDX-License-Identifier: Apache-2.0

// The binding parent needs the numeric project number. Read only when tags are set.
data "google_project" "resource_manager_tags" {
  count = length(var.resource_manager_tags) > 0 ? 1 : 0

  project_id = var.project
}

// Cloud SQL tag bindings use the sqladmin service domain, not the
// cloudsql.googleapis.com name that Cloud Asset Inventory reports, and they are
// location scoped to the instance region.
resource "google_tags_location_tag_binding" "primary" {
  for_each = var.resource_manager_tags

  parent    = "//sqladmin.googleapis.com/projects/${data.google_project.resource_manager_tags[0].number}/instances/${google_sql_database_instance.this.name}"
  tag_value = each.value
  location  = var.region
}

locals {
  replica_tag_bindings = {
    for pair in setproduct(var.read_replica_regions, keys(var.resource_manager_tags)) :
    "${pair[0]}/${pair[1]}" => {
      region    = pair[0]
      tag_value = var.resource_manager_tags[pair[1]]
    }
  }
}

resource "google_tags_location_tag_binding" "replicas" {
  for_each = local.replica_tag_bindings

  parent    = "//sqladmin.googleapis.com/projects/${data.google_project.resource_manager_tags[0].number}/instances/${google_sql_database_instance.replicas[each.value.region].name}"
  tag_value = each.value.tag_value
  location  = each.value.region
}
