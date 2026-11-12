# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0
#
# Upgrading from a release that created the placeholder with secret_data.
# The legacy fixture applies the old address into shared state, and the real
# module then plans against that state. terraform test can't assert on a
# planned action, so this pins that the upgrade plans and adds no version,
# not that the removed block forgets rather than destroys; -verbose shows
# the forget.

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
  authorized-adder      = "serviceAccount:adder@fixture-project.iam.gserviceaccount.com"
  service-account       = "reader@fixture-project.iam.gserviceaccount.com"
  notification-channels = []
  team                  = "fixture"
}

run "legacy_placeholder" {
  state_key = "upgrade"
  module {
    source = "./tests/legacy_placeholder"
  }
}

run "upgrade_adds_no_version" {
  state_key = "upgrade"
  command   = plan
  variables {
    create_placeholder_version = true
  }

  assert {
    condition     = length(google_secret_manager_secret_version.bootstrap) == 0
    error_message = "upgrading a caller that set create_placeholder_version must not add a version: it would become latest"
  }
}
