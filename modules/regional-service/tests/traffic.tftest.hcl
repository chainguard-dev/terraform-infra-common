# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0
# Run in CI by .github/workflows/tf-module-tests.yaml.
#

# Plan-only tests that pin whether Terraform declares the traffic split.
#
# Cloud Run's traffic field is Optional+Computed: an undeclared block leaves
# whatever the API holds, so an out-of-band `gcloud run services
# update-traffic` pin survives every later apply and new revisions deploy
# without ever serving. manage_traffic is the opt-in that takes ownership
# back. These tests fail if the default starts declaring a split (which would
# silently revert deliberate pins fleet-wide) or if the opt-in stops
# declaring 100% to latest.
#
# Mock providers keep this fully offline: no credentials, no state.

mock_provider "google-beta" {}

mock_provider "google" {
  mock_data "google_project" {
    defaults = {
      number = "123456789"
    }
  }
}

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
      image = "cgr.dev/chainguard/static:latest"
      ports = [{ container_port = 8080 }]
    }
  }
}

# Consumers that say nothing about traffic keep the pre-existing behaviour:
# no traffic block, so the API's split is left alone.
run "default_declares_no_traffic_block" {
  command = plan

  assert {
    condition     = length(google_cloud_run_v2_service.this["us-central1"].traffic) == 0
    error_message = "a traffic block rendered with manage_traffic unset; the default must leave the split unmanaged"
  }
}

# Opting in declares the whole split, which is what reverts a manual pin.
run "manage_traffic_declares_all_traffic_to_latest" {
  command = plan

  variables {
    manage_traffic = true
  }

  assert {
    condition     = length(google_cloud_run_v2_service.this["us-central1"].traffic) == 1
    error_message = "manage_traffic = true must render exactly one traffic target; more than one leaves room for a pinned revision to keep serving"
  }

  assert {
    condition     = google_cloud_run_v2_service.this["us-central1"].traffic[0].type == "TRAFFIC_TARGET_ALLOCATION_TYPE_LATEST"
    error_message = "manage_traffic = true rendered a traffic target that is not TRAFFIC_TARGET_ALLOCATION_TYPE_LATEST"
  }

  assert {
    condition     = google_cloud_run_v2_service.this["us-central1"].traffic[0].percent == 100
    error_message = "manage_traffic = true rendered a traffic target at less than 100%; the remainder would stay on the pinned revision"
  }

  assert {
    condition     = google_cloud_run_v2_service.this["us-central1"].traffic[0].revision == null || google_cloud_run_v2_service.this["us-central1"].traffic[0].revision == ""
    error_message = "a LATEST traffic target must not name a revision"
  }
}
