# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

mock_provider "google" {}

variables {
  project_id      = "fixture-project"
  region          = "us-east1"
  name            = "fixture-private-service"
  service-account = "dispatcher@fixture-project.iam.gserviceaccount.com"
}

run "service_invocation_grant" {
  command = plan

  assert {
    condition = jsonencode(output.invocation_grant) == jsonencode({
      project   = "fixture-project"
      location  = "us-east1"
      name      = "fixture-private-service"
      role      = "roles/run.invoker"
      member    = "serviceAccount:dispatcher@fixture-project.iam.gserviceaccount.com"
      condition = []
    })
    error_message = "The output must describe the managed service invocation grant."
  }
}
