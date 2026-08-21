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
  mock_resource "google_compute_instance" {
    override_during = plan
    defaults = {
      instance_id = "987654321"
    }
  }
}

variables {
  name           = "fixture"
  project_id     = "fixture-project"
  zone           = "us-central1-a"
  network        = "fixture-network"
  subnetwork     = "fixture-subnetwork"
  dev_principals = []
  team           = "fixture"
}

run "resource_manager_tags_default_to_empty" {
  command = plan
  assert {
    condition     = length(google_tags_location_tag_binding.this) == 0
    error_message = "default resource_manager_tags must create no tag bindings"
  }
}

run "resource_manager_tags_bind_compute_zone" {
  command = plan
  variables {
    resource_manager_tags = { "tagKeys/123" = "tagValues/456" }
  }
  assert {
    condition     = google_tags_location_tag_binding.this["tagKeys/123"].location == "us-central1-a"
    error_message = "Compute tag binding must use the instance zone"
  }
  assert {
    condition     = google_tags_location_tag_binding.this["tagKeys/123"].parent == "//compute.googleapis.com/projects/123456789/zones/us-central1-a/instances/987654321"
    error_message = "Compute tag binding must use the documented full resource name"
  }
}
