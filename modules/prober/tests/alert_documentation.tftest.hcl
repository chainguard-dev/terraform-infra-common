# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0
#
# Cloud Monitoring rejects alert documentation over its limits only at apply,
# so a mocked plan would accept it; the module validates the limits at plan.

mock_provider "google" {}
mock_provider "google-beta" {}
mock_provider "ko" {}
mock_provider "cosign" {}
mock_provider "random" {}

variables {
  project_id      = "fixture-project"
  name            = "fixture"
  service_account = "fixture@fixture-project.iam.gserviceaccount.com"
  importpath      = "example.com/fixture/cmd/prober"
  working_dir     = "."
  team            = "fixture"
  regions = {
    "us-central1" = {
      network = "projects/fixture-project/global/networks/fixture"
      subnet  = "projects/fixture-project/regions/us-central1/subnetworks/fixture"
    }
  }
  notification_channels = []
  enable_alert          = true
}

run "three_links_plan_cleanly" {
  command = plan

  variables {
    alert_links = [
      { display_name = "Prober logs", url = "https://example.com/logs" },
      { display_name = "Dashboard", url = "https://example.com/dashboard" },
      { display_name = "Source", url = "https://example.com/source" },
    ]
  }

  assert {
    condition     = length(google_monitoring_alert_policy.uptime_alert[0].documentation[0].links) == 3
    error_message = "every supplied link should reach the alert documentation."
  }
}

run "four_links_are_rejected" {
  command = plan

  variables {
    alert_links = [
      { display_name = "Prober logs", url = "https://example.com/logs" },
      { display_name = "Dashboard", url = "https://example.com/dashboard" },
      { display_name = "Source", url = "https://example.com/source" },
      { display_name = "Runbook", url = "https://example.com/runbook" },
    ]
  }

  expect_failures = [var.alert_links]
}

run "empty_display_name_is_rejected" {
  command = plan

  variables {
    alert_links = [{ display_name = "", url = "https://example.com/runbook" }]
  }

  expect_failures = [var.alert_links]
}

run "display_name_over_63_characters_is_rejected" {
  command = plan

  variables {
    alert_links = [{ display_name = replace(format("%64s", ""), " ", "a"), url = "https://example.com/runbook" }]
  }

  expect_failures = [var.alert_links]
}

run "url_over_2083_characters_is_rejected" {
  command = plan

  variables {
    alert_links = [{ display_name = "Runbook", url = "https://example.com/${replace(format("%2064s", ""), " ", "a")}" }]
  }

  expect_failures = [var.alert_links]
}

run "description_over_8192_characters_is_rejected" {
  command = plan

  variables {
    alert_description = replace(format("%8193s", ""), " ", "a")
  }

  expect_failures = [var.alert_description]
}
