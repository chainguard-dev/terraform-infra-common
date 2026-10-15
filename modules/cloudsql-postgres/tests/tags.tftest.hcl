# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0
#
# Run in CI by .github/workflows/tf-module-tests.yaml.

mock_provider "google" {}

variables {
  name    = "fixture"
  project = "fixture-project"
  region  = "us-central1"
  network = "projects/fixture-project/global/networks/fixture"
  team    = "fixture"
  edition = "ENTERPRISE"
}

run "resource_manager_tags_default_to_empty" {
  command = plan
  assert {
    condition     = length(google_tags_location_tag_binding.primary) == 0
    error_message = "default resource_manager_tags must create no tag bindings"
  }
}

# Cloud SQL tag bindings use the sqladmin service domain, not the
# cloudsql.googleapis.com name that Cloud Asset Inventory reports. Pin it so a
# well-intentioned "fix" to the asset-inventory spelling cannot land silently.
run "resource_manager_tags_bind_cloudsql_via_sqladmin" {
  command = plan
  variables {
    resource_manager_tags = { "tagKeys/123" = "tagValues/456" }
  }
  assert {
    condition     = google_tags_location_tag_binding.primary["tagKeys/123"].location == "us-central1"
    error_message = "Cloud SQL tag binding must be located to the instance region"
  }
  assert {
    condition     = google_tags_location_tag_binding.primary["tagKeys/123"].parent == "//sqladmin.googleapis.com/projects/fixture-project/instances/fixture"
    error_message = "Cloud SQL tag binding must use the sqladmin full resource name with the project ID, the form the API returns"
  }
}

# The API returns a Cloud SQL binding's parent with the project ID. A parent
# built from the project number differs from state on every plan, forces a
# replacement, and the create fails because the binding already exists. Replica
# bindings follow the same rule.
run "resource_manager_tags_bind_replicas_by_project_id" {
  command = plan
  variables {
    resource_manager_tags = { "tagKeys/123" = "tagValues/456" }
    read_replica_regions  = ["us-east4"]
    backup_enabled        = true
  }
  assert {
    condition     = google_tags_location_tag_binding.replicas["us-east4/tagKeys/123"].parent == "//sqladmin.googleapis.com/projects/fixture-project/instances/${google_sql_database_instance.replicas["us-east4"].name}"
    error_message = "Cloud SQL replica tag binding must name the project by ID"
  }
  assert {
    condition     = google_tags_location_tag_binding.replicas["us-east4/tagKeys/123"].location == "us-east4"
    error_message = "Cloud SQL replica tag binding must be located to the replica region"
  }
}
