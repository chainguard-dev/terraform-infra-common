# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

mock_provider "google" {}
mock_provider "google-beta" {}
mock_provider "ko" {}
mock_provider "cosign" {}

override_resource {
  target          = google_service_account.invoker
  override_during = plan
  values = {
    name   = "projects/fixture-project/serviceAccounts/scheduler@fixture-project.iam.gserviceaccount.com"
    email  = "scheduler@fixture-project.iam.gserviceaccount.com"
    member = "serviceAccount:scheduler@fixture-project.iam.gserviceaccount.com"
  }
}

variables {
  project_id      = "fixture-project"
  name            = "fixture-job"
  service_account = "worker@fixture-project.iam.gserviceaccount.com"
  team            = "fixture"
  regions = {
    us-central1 = {}
    us-east1    = {}
  }
  regional-cronspec = {
    us-central1 = { schedule = "0 * * * *" }
    us-east1    = { schedule = "30 * * * *" }
  }
  invokers = ["group:operators@example.com", "user:operator@example.com"]
}

run "regional_job_invocation_grants" {
  command = plan

  assert {
    condition = jsonencode(output.invocation_grants) == jsonencode({
      for region in ["us-central1", "us-east1"] : region => {
        project  = "fixture-project"
        location = region
        name     = "fixture-job"
        role     = "roles/run.invoker"
        members = toset([
          "group:operators@example.com",
          "user:operator@example.com",
          "serviceAccount:scheduler@fixture-project.iam.gserviceaccount.com",
        ])
        condition = []
      }
    })
    error_message = "Every regional job grant must include the scheduler and exactly the configured additional invokers."
  }
}

run "removed_operator_grants" {
  command = plan

  variables {
    invokers = []
  }

  assert {
    condition = jsonencode(output.invocation_grants) == jsonencode({
      for region in ["us-central1", "us-east1"] : region => {
        project   = "fixture-project"
        location  = region
        name      = "fixture-job"
        role      = "roles/run.invoker"
        members   = toset(["serviceAccount:scheduler@fixture-project.iam.gserviceaccount.com"])
        condition = []
      }
    })
    error_message = "Removing additional invokers must leave only the dedicated scheduler account."
  }
}
