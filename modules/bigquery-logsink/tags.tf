// Copyright 2026 Chainguard, Inc.
// SPDX-License-Identifier: Apache-2.0

// The binding parent needs the numeric project number. Read only when tags are set.
data "google_project" "resource_manager_tags" {
  count = length(var.resource_manager_tags) > 0 ? 1 : 0

  project_id = var.project_id
}

// BigQuery tag bindings are location scoped to the dataset location.
// hashicorp/terraform-provider-google#18254 breaks multi-region locations, so a
// validation on var.resource_manager_tags rejects that combination at plan.
resource "google_tags_location_tag_binding" "dataset" {
  for_each = var.resource_manager_tags

  parent    = "//bigquery.googleapis.com/projects/${data.google_project.resource_manager_tags[0].number}/datasets/${google_bigquery_dataset.this.dataset_id}"
  tag_value = each.value
  location  = lower(var.location)
}
