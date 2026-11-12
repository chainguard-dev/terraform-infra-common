# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0
#
# Run in CI by .chainguard/ci/tf-module-tests.yaml.
#
# The provider reads a version's payload on refresh unless the version was
# written through secret_data_wo, so a plan identity without secretAccessor
# can refresh only a write-only version.

mock_provider "google" {}

variables {
  project_id            = "fixture-project"
  name                  = "fixture"
  data                  = "fixture-data"
  service-account       = "fixture@fixture-project.iam.gserviceaccount.com"
  notification-channels = []
  team                  = "fixture"
}

run "data_is_write_only" {
  command = plan

  assert {
    condition     = google_secret_manager_secret_version.data.secret_data == null
    error_message = "the data must be written through secret_data_wo, so that refreshing the version never reads its payload"
  }
  assert {
    condition     = google_secret_manager_secret_version.data.secret_data_wo_version == sha256("fixture-data")
    error_message = "the write-only version must follow the data, so that changed data adds a new version"
  }
}
