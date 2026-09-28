# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0
#
# Run in CI by .github/workflows/tf-module-tests.yaml.
#
# timeout is a duration string. A bare number type-converts to a string and
# plans cleanly against mock providers, then fails at apply on the provider's
# "^[0-9]+(?:\.[0-9]{1,9})?s$" check, so the module validates it at plan.

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
    "us-central1" = {
      network = "projects/fixture-project/global/networks/fixture"
      subnet  = "projects/fixture-project/regions/us-central1/subnetworks/fixture"
    }
  }
  regional-cronspec = {
    "us-central1" = { schedule = "0 * * * *" }
  }
}

run "duration_plans_cleanly" {
  command = plan

  variables {
    timeout = "240s"
  }
}

run "bare_number_is_rejected" {
  command = plan

  variables {
    timeout = 240
  }

  expect_failures = [var.timeout]
}
