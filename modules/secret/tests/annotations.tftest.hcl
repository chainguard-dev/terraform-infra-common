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
  project_id            = "fixture-project"
  name                  = "fixture"
  authorized-adder      = "serviceAccount:fixture@fixture-project.iam.gserviceaccount.com"
  service-account       = "fixture@fixture-project.iam.gserviceaccount.com"
  notification-channels = []
  team                  = "fixture"
}

# Existing callers pass no annotations; the secret must plan none, so they see
# no diff.
run "annotations_default_to_empty" {
  command = plan
  assert {
    condition     = length(google_secret_manager_secret.this.annotations) == 0
    error_message = "default annotations must set none on the secret"
  }
}

# The redis module publishes connection details through this input.
run "annotations_reach_the_secret" {
  command = plan
  variables {
    annotations = { "redis-host" = "10.0.0.3", "redis-port" = "6378" }
  }
  assert {
    condition     = google_secret_manager_secret.this.annotations == tomap({ "redis-host" = "10.0.0.3", "redis-port" = "6378" })
    error_message = "annotations must be set on the secret as given"
  }
}
