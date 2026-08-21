# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0
#
# Run in CI by .github/workflows/tf-module-tests.yaml.

mock_provider "google" {
  mock_resource "google_container_cluster" {
    override_during = plan
    defaults = {
      id       = "projects/fixture-project/locations/us-central1/clusters/fixture"
      location = "us-central1"
    }
  }
}

mock_provider "google-beta" {}

variables {
  name       = "fixture"
  project    = "fixture-project"
  network    = "fixture-network"
  region     = "us-central1"
  team       = "fixture"
  subnetwork = "fixture-subnetwork"
  pools      = {}
}

run "resource_manager_tags_default_to_empty" {
  command = plan

  assert {
    condition     = length(google_tags_location_tag_binding.cluster) == 0
    error_message = "default resource_manager_tags must create no tag bindings"
  }
}

run "resource_manager_tags_bind_cluster" {
  command = plan

  variables {
    resource_manager_tags = { "tagKeys/123" = "tagValues/456" }
  }

  assert {
    condition     = google_tags_location_tag_binding.cluster["tagKeys/123"].parent == "//container.googleapis.com/projects/fixture-project/locations/us-central1/clusters/fixture"
    error_message = "GKE tag binding must use the documented cluster full resource name"
  }

  assert {
    condition     = google_tags_location_tag_binding.cluster["tagKeys/123"].location == "us-central1"
    error_message = "GKE tag binding must use the cluster location"
  }

  assert {
    condition     = google_tags_location_tag_binding.cluster["tagKeys/123"].tag_value == "tagValues/456"
    error_message = "GKE tag binding must preserve the requested tag value"
  }
}
