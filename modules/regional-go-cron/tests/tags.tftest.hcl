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
mock_provider "google-beta" {}
mock_provider "ko" {}
mock_provider "cosign" {}

variables {
  project_id      = "fixture-project"
  name            = "fixture"
  service_account = "fixture@fixture-project.iam.gserviceaccount.com"
  team            = "fixture"
  regions = {
    us-central1 = {}
  }
  regional-cronspec = {
    us-central1 = { schedule = "0 * * * *" }
  }
}

run "resource_manager_tags_default_to_empty" {
  command = plan
  assert {
    condition     = length(google_tags_location_tag_binding.this) == 0
    error_message = "default resource_manager_tags must create no tag bindings"
  }
}

run "resource_manager_tags_bind_every_region_tag_pair" {
  command = plan
  variables {
    regions = {
      us-central1 = {}
      us-east1    = {}
    }
    regional-cronspec = {
      us-central1 = { schedule = "0 * * * *" }
      us-east1    = { schedule = "30 * * * *" }
    }
    resource_manager_tags = {
      "tagKeys/123" = "tagValues/456"
      "tagKeys/789" = "tagValues/012"
    }
  }
  assert {
    condition     = length(google_tags_location_tag_binding.this) == 4
    error_message = "two tags on two regions must create four location tag bindings"
  }
  assert {
    condition     = google_tags_location_tag_binding.this["us-east1/tagKeys/789"].location == "us-east1"
    error_message = "regional job tag binding must preserve the region for each cartesian-product pair"
  }
  assert {
    condition     = google_tags_location_tag_binding.this["us-east1/tagKeys/789"].parent == "//run.googleapis.com/projects/123456789/locations/us-east1/jobs/fixture"
    error_message = "Cloud Run tag binding must use the numeric project number"
  }
}
