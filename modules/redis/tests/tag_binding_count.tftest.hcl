# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

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

run "tag_binding_count_defaults_to_zero" {
  command = plan
  assert {
    condition     = output.tag_binding_count == 0
    error_message = "tag_binding_count must be 0 without resource_manager_tags"
  }
}

run "tag_binding_count_matches_tags" {
  command = plan
  variables {
    resource_manager_tags = {
      "tagKeys/123" = "tagValues/456"
      "tagKeys/789" = "tagValues/012"
    }
  }
  assert {
    condition     = output.tag_binding_count == 2
    error_message = "tag_binding_count must equal the number of resource_manager_tags"
  }
}
