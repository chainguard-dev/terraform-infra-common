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
      source = {
        working_dir = "."
        importpath  = "example.com/fixture"
      }
    }
  }
}

run "jobs_scrape_every_10_seconds" {
  command = plan
  assert {
    condition = strcontains(one(flatten([
      for container in google_cloud_run_v2_job.this["us-central1"].template[0].template[0].containers : [
        for e in container.env : e.value if e.name == "OTEL_CONFIG"
      ]
    ])), "        scrape_interval: 10s\n")
    error_message = "Jobs size their end-of-run hold for the final scrape to a 10s scrape interval."
  }
}
