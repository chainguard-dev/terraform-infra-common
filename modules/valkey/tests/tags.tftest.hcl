# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0
#
# Run in CI by .github/workflows/tf-module-tests.yaml.

mock_provider "google" {
  mock_data "google_project" {
    defaults = {
      number = "123456789"
    }
  }
}

variables {
  project_id = "fixture-project"
  name       = "fixture"
  region     = "us-central1"
  network    = "projects/fixture-project/global/networks/fixture"
  team       = "fixture"
}

run "resource_manager_tags_default_to_empty" {
  command = plan
  assert {
    condition     = length(google_tags_location_tag_binding.valkey) == 0
    error_message = "default resource_manager_tags must create no tag bindings"
  }
}

run "resource_manager_tags_bind_memorystore" {
  command = plan
  variables {
    resource_manager_tags = { "tagKeys/123" = "tagValues/456" }
  }
  assert {
    condition     = google_tags_location_tag_binding.valkey["tagKeys/123"].location == "us-central1"
    error_message = "Memorystore tag binding must use the instance region"
  }
  assert {
    condition     = google_tags_location_tag_binding.valkey["tagKeys/123"].parent == "//memorystore.googleapis.com/projects/123456789/locations/us-central1/instances/fixture"
    error_message = "Memorystore tag binding must use the numeric project number in the verified full resource name"
  }
}
