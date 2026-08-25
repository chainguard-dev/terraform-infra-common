# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

# Plan-only guard on the rendered otel sidecar resource clause.
#
# This module owns the sidecar container, so its otel_resources default is
# what direct consumers inherit. When that default was null the sidecar
# rendered with no resources block and silently fell back to the Cloud Run
# platform allocation, which no consumer diff shows. This run pins the
# rendered limits so a default change has to be deliberate.
#
# Mock providers keep this fully offline: no credentials, no state.

mock_provider "google" {}

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

run "default_otel_resources_render_explicit_sidecar_limits" {
  command = plan

  variables {
    enable_otel_sidecar = true
  }

  assert {
    condition = (
      google_cloud_run_v2_service.this["us-central1"].template[0].containers[1].resources[0].limits.cpu == "250m" &&
      google_cloud_run_v2_service.this["us-central1"].template[0].containers[1].resources[0].limits.memory == "512Mi"
    )
    error_message = "otel sidecar rendered ${jsonencode(google_cloud_run_v2_service.this["us-central1"].template[0].containers[1].resources)} instead of the 250m/512Mi default"
  }
}
