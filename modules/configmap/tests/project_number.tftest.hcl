# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0
#
# Run in CI by .github/workflows/tf-module-tests.yaml.
#
# The mocked lookup number differs from the supplied one, so a leaked
# google_project lookup changes the asserted parent.

mock_provider "google" {
  mock_data "google_project" {
    defaults = {
      number = "123456789"
    }
  }
}

variables {
  project_id            = "fixture-project"
  name                  = "fixture"
  data                  = "fixture-data"
  service-account       = "fixture@fixture-project.iam.gserviceaccount.com"
  notification-channels = []
  team                  = "fixture"
}

run "project_number_skips_lookup_and_is_used" {
  command = plan

  variables {
    project_number        = "987654321"
    resource_manager_tags = { "tagKeys/123" = "tagValues/456" }
  }

  assert {
    condition     = length(data.google_project.project) == 0
    error_message = "project_number must skip the google_project data source"
  }
  assert {
    condition     = google_tags_tag_binding.this["tagKeys/123"].parent == "//secretmanager.googleapis.com/projects/987654321/secrets/fixture"
    error_message = "tag binding parent must use the supplied project_number"
  }
}

run "null_project_number_falls_back_to_lookup" {
  command = plan

  variables {
    resource_manager_tags = { "tagKeys/123" = "tagValues/456" }
  }

  assert {
    condition     = length(data.google_project.project) == 1
    error_message = "a null project_number must read the google_project data source"
  }
  assert {
    condition     = google_tags_tag_binding.this["tagKeys/123"].parent == "//secretmanager.googleapis.com/projects/123456789/secrets/fixture"
    error_message = "tag binding parent must use the looked-up number when project_number is null"
  }
}

run "empty_tags_bind_nothing_and_skip_lookup" {
  command = plan

  assert {
    condition     = length(google_tags_tag_binding.this) == 0
    error_message = "empty tags must create no tag bindings"
  }
  assert {
    condition     = length(data.google_project.project) == 0
    error_message = "empty tags must not read the google_project data source"
  }
}

run "non_numeric_project_number_is_rejected" {
  command = plan

  variables {
    project_number = "fixture-project"
  }

  expect_failures = [var.project_number]
}
