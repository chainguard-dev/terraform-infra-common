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
  project_id               = "fixture-project"
  name                     = "fixture"
  team                     = "fixture"
  region                   = "us-central1"
  zone                     = "us-central1-a"
  secret_accessor_sa_email = "fixture@fixture-project.iam.gserviceaccount.com"
  notification_channels    = []
  secret_version_adder     = "serviceAccount:fixture@fixture-project.iam.gserviceaccount.com"
}

run "resource_manager_tags_default_to_empty" {
  command = plan
  assert {
    condition     = length(google_tags_location_tag_binding.this) == 0
    error_message = "default resource_manager_tags must create no tag bindings"
  }
}

run "resource_manager_tags_bind_redis_region" {
  command = plan
  variables {
    resource_manager_tags = { "tagKeys/123" = "tagValues/456" }
  }
  assert {
    condition     = google_tags_location_tag_binding.this["tagKeys/123"].location == "us-central1"
    error_message = "Redis tag binding must use the instance region"
  }
  assert {
    condition     = google_tags_location_tag_binding.this["tagKeys/123"].parent == "//redis.googleapis.com/projects/123456789/locations/us-central1/instances/fixture"
    error_message = "Redis tag binding must use the documented full resource name"
  }
}
