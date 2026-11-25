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
  project_id      = "fixture-project"
  name            = "fixture"
  location        = "us-central1"
  service_account = "serviceAccount:fixture@fixture-project.iam.gserviceaccount.com"
}

run "resource_manager_tags_default_to_empty" {
  command = plan
  assert {
    condition     = length(google_tags_location_tag_binding.attestations) == 0
    error_message = "default resource_manager_tags must create no tag bindings"
  }
}

run "resource_manager_tags_bind_artifact_registry" {
  command = plan
  variables {
    resource_manager_tags = { "tagKeys/123" = "tagValues/456" }
  }
  assert {
    condition     = google_tags_location_tag_binding.attestations["tagKeys/123"].location == "us-central1"
    error_message = "Artifact Registry tag binding must use the repository location"
  }
  assert {
    condition     = google_tags_location_tag_binding.attestations["tagKeys/123"].parent == "//artifactregistry.googleapis.com/projects/123456789/locations/us-central1/repositories/fixture"
    error_message = "Artifact Registry tag binding must use the numeric project number in the verified full resource name"
  }
}

run "malformed_tags_are_rejected" {
  command = plan
  variables {
    resource_manager_tags = { "tagKeys/nope" = "tagValues/nope" }
  }
  expect_failures = [var.resource_manager_tags]
}
