# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

mock_provider "google" {
  mock_data "google_project" {
    defaults = { number = "123456789" }
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
  regions         = { us-central1 = {} }
  regional-cronspec = {
    us-central1 = { schedule = "0 0 9 1 *", paused = true }
  }
  containers = {
    this = {
      name = "build"
      source = {
        working_dir = "."
        importpath  = "example.com/fixture"
      }
    }
  }
}

run "explicit_application_container_name_with_sidecar" {
  command = plan
  assert {
    condition = length([
      for container in google_cloud_run_v2_job.this["us-central1"].template[0].template[0].containers : container
      if container.name == "build"
    ]) == 1
    error_message = "Execution overrides must address exactly one application container by its explicit name."
  }
  assert {
    condition     = length(google_cloud_run_v2_job.this["us-central1"].template[0].template[0].containers) == 2
    error_message = "The named application must retain its observability sidecar."
  }
}
