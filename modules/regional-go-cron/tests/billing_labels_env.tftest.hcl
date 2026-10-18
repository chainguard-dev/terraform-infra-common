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
  team            = "fixture-team"
  product         = "fixture-product"
  regions         = { us-central1 = {} }
  regional-cronspec = {
    us-central1 = { schedule = "0 0 9 1 *", paused = true }
  }
  containers = {
    this = {
      source = {
        working_dir = "."
        importpath  = "example.com/fixture"
      }
    }
  }
}

run "job_sets_team_and_product_env" {
  command = plan
  assert {
    condition = contains(flatten([
      for container in google_cloud_run_v2_job.this["us-central1"].template[0].template[0].containers : [
        for e in container.env : e.value if e.name == "CHAINGUARD_TEAM"
      ]
    ]), "fixture-team")
    error_message = "Jobs must set CHAINGUARD_TEAM from the team input."
  }
  assert {
    condition = contains(flatten([
      for container in google_cloud_run_v2_job.this["us-central1"].template[0].template[0].containers : [
        for e in container.env : e.value if e.name == "CHAINGUARD_PRODUCT"
      ]
    ]), "fixture-product")
    error_message = "Jobs must set CHAINGUARD_PRODUCT from the product input."
  }
}
