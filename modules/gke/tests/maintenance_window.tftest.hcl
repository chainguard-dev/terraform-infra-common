# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0
#
# Run in CI by .github/workflows/tf-module-tests.yaml.

mock_provider "google" {
  mock_resource "google_container_cluster" {
    override_during = plan
    defaults = {
      id       = "projects/fixture-project/locations/us-central1/clusters/fixture"
      location = "us-central1"
    }
  }
}

mock_provider "google-beta" {}

variables {
  name       = "fixture"
  project    = "fixture-project"
  network    = "fixture-network"
  region     = "us-central1"
  team       = "fixture"
  subnetwork = "fixture-subnetwork"
  pools      = {}
}

run "no_window_by_default" {
  command = plan

  assert {
    condition     = length(google_container_cluster.this.maintenance_policy) == 0
    error_message = "an unset maintenance_recurring_window must render no maintenance_policy block"
  }
}

run "window_rides_through" {
  command = plan

  variables {
    maintenance_recurring_window = {
      start_time = "2026-01-05T05:00:00Z"
      end_time   = "2026-01-05T12:00:00Z"
      recurrence = "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR"
    }
  }

  assert {
    condition     = google_container_cluster.this.maintenance_policy[0].recurring_window[0].start_time == "2026-01-05T05:00:00Z"
    error_message = "recurring_window.start_time must carry start_time through unchanged"
  }

  assert {
    condition     = google_container_cluster.this.maintenance_policy[0].recurring_window[0].end_time == "2026-01-05T12:00:00Z"
    error_message = "recurring_window.end_time must carry end_time through unchanged"
  }

  assert {
    condition     = google_container_cluster.this.maintenance_policy[0].recurring_window[0].recurrence == "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR"
    error_message = "recurring_window.recurrence must carry recurrence through unchanged"
  }
}
