# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0
#
# Run in CI by .github/workflows/tf-module-tests.yaml.

mock_provider "google" {}
mock_provider "random" {
  mock_resource "random_string" {
    override_during = plan
    defaults = {
      result = "abc123"
    }
  }
}

variables {
  project_id = "fixture-project"
  name       = "fixture"
  location   = "US"
}

run "resource_manager_tags_default_to_empty" {
  command = plan
  assert {
    condition     = length(google_tags_location_tag_binding.this) == 0
    error_message = "default resource_manager_tags must create no tag bindings"
  }
}

run "resource_manager_tags_bind_bucket_location" {
  command = plan
  variables {
    resource_manager_tags = { "tagKeys/123" = "tagValues/456" }
  }
  assert {
    condition     = google_tags_location_tag_binding.this["tagKeys/123"].location == "us"
    error_message = "bucket tag binding location must be lowercase"
  }
  assert {
    condition     = google_tags_location_tag_binding.this["tagKeys/123"].parent == "//storage.googleapis.com/projects/_/buckets/fixture-abc123"
    error_message = "bucket tag binding must use the documented full resource name"
  }
}

run "labels_default_to_empty" {
  command = plan
  assert {
    condition     = length(google_storage_bucket.status.labels) == 0
    error_message = "a caller passing no labels must get an unlabelled bucket"
  }
}

run "team_and_product_are_not_labels" {
  command = plan
  variables {
    team    = "containers"
    product = "containers"
    labels  = { component = "watermarks" }
  }
  assert {
    condition     = google_storage_bucket.status.labels == tomap({ component = "watermarks" })
    error_message = "buckets support tags, so team and product must not become labels; extra labels still reach the bucket"
  }
}
