# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

mock_provider "google" {}

variables {
  project_id            = "fixture-project"
  name                  = "fixture-ci"
  wif-pool              = "projects/123456789/locations/global/workloadIdentityPools/fixture-pool"
  repository            = "fixture-org/fixture-repo"
  refspec               = "refs/heads/main"
  workflow_ref          = ".github/workflows/deploy.yaml"
  notification_channels = []
}

run "enabled_by_default" {
  command = plan

  assert {
    condition     = google_service_account.this.disabled == false
    error_message = "With disabled unset, the service account must stay enabled."
  }
}

run "disabled_when_set" {
  command = plan

  variables {
    disabled = true
  }

  assert {
    condition     = google_service_account.this.disabled == true
    error_message = "disabled = true must reach google_service_account.this."
  }
}
