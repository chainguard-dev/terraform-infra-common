# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0
#
# Run in CI by .chainguard/ci/tf-module-tests.yaml.
#
# cron forwards otel_collector_image to regional-go-cron with its own
# default, so this checks that the image a cron caller gets by default is
# pinned by digest and keeps its tag for digestabot.
# ../regional-go-cron/tests/otel_collector_image.tftest.hcl says why.

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
  project_id            = "fixture-project"
  name                  = "fixture"
  schedule              = "0 * * * *"
  service_account       = "fixture@fixture-project.iam.gserviceaccount.com"
  working_dir           = "."
  importpath            = "example.com/fixture/cmd/app"
  team                  = "fixture"
  notification_channels = []
}

run "default_sidecar_image_is_pinned_by_tag_and_digest" {
  command = plan

  assert {
    condition     = can(regex("^[^@]+:[^@/:]+@sha256:[0-9a-f]{64}$", nonsensitive(module.impl.containers[var.region][1].image)))
    error_message = "The cron otel sidecar default must be repository:tag@sha256:<digest>."
  }
}
