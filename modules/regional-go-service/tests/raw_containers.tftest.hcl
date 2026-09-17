# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

# Prebuilt images must not replace source-built, signed containers.
# Mock providers keep this plan-only test offline.
mock_provider "ko" {}
mock_provider "cosign" {}
mock_provider "google" {}
mock_provider "google-beta" {}

variables {
  project_id = "fixture-project"
  name       = "fixture"
  regions = {
    "us-central1" = {
      network = "projects/fixture-project/global/networks/fixture"
      subnet  = "projects/fixture-project/regions/us-central1/subnetworks/fixture"
    }
  }
  service_account       = "fixture@fixture-project.iam.gserviceaccount.com"
  notification_channels = []
  team                  = "fixture"
  containers = {
    "main" = {
      source = {
        working_dir = "."
        importpath  = "example.com/fixture/cmd/app"
      }
      ports = [{ container_port = 8080 }]
    }
  }
}

run "prebuilt_image_cannot_replace_signed_container" {
  command = plan
  variables {
    raw_containers = {
      main = {
        image = "example.com/helper:fixture"
        ports = [{ container_port = 8080 }]
      }
    }
  }
  expect_failures = [cosign_sign.this["main"]]
}
