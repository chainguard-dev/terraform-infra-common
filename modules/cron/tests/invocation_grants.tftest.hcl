# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

mock_provider "google" {}
mock_provider "google-beta" {}
mock_provider "ko" {}
mock_provider "cosign" {}

override_resource {
  target          = module.impl.google_service_account.invoker
  override_during = plan
  values = {
    name   = "projects/fixture-project/serviceAccounts/scheduler@fixture-project.iam.gserviceaccount.com"
    email  = "scheduler@fixture-project.iam.gserviceaccount.com"
    member = "serviceAccount:scheduler@fixture-project.iam.gserviceaccount.com"
  }
}

variables {
  project_id            = "fixture-project"
  name                  = "fixture-recovery"
  region                = "us-east1"
  schedule              = "0 * * * *"
  service_account       = "worker@fixture-project.iam.gserviceaccount.com"
  working_dir           = "."
  importpath            = "example.com/fixture/cmd/recovery"
  team                  = "fixture"
  notification_channels = []
  invokers              = ["user:operator@example.com"]
}

run "single_region_job_invocation_grants" {
  command = plan

  assert {
    condition = jsonencode(output.invocation_grants) == jsonencode({
      us-east1 = {
        project  = "fixture-project"
        location = "us-east1"
        name     = "fixture-recovery-cron"
        role     = "roles/run.invoker"
        members = toset([
          "user:operator@example.com",
          "serviceAccount:scheduler@fixture-project.iam.gserviceaccount.com",
        ])
        condition = []
      }
    })
    error_message = "The cron wrapper must export the actual regional job invocation binding."
  }
}
