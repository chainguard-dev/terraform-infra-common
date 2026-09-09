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
  pools = {
    plain = {}
    spot-alias = {
      spot = true
    }
    spot = {
      provisioning_model = "spot"
    }
    flex = {
      provisioning_model = "flex-start"
      max_run_duration   = "57600s"
      min_node_count     = 0
      max_node_count     = 4
    }
  }
}

run "flex_start_pool_renders_the_gke_requirements" {
  command = plan

  assert {
    condition     = google_container_node_pool.pools["flex"].node_config[0].flex_start == true
    error_message = "a flex-start pool must set node_config.flex_start"
  }

  assert {
    condition     = google_container_node_pool.pools["flex"].node_config[0].spot == false
    error_message = "a flex-start pool must not be spot"
  }

  assert {
    condition     = google_container_node_pool.pools["flex"].node_config[0].max_run_duration == "57600s"
    error_message = "a flex-start pool must carry max_run_duration through unchanged"
  }

  assert {
    condition     = google_container_node_pool.pools["flex"].autoscaling[0].location_policy == "ANY"
    error_message = "a flex-start pool must use location policy ANY"
  }

  assert {
    condition     = google_container_node_pool.pools["flex"].management[0].auto_repair == false
    error_message = "a flex-start pool must have auto-repair off"
  }

  assert {
    condition     = google_container_node_pool.pools["flex"].node_config[0].reservation_affinity[0].consume_reservation_type == "NO_RESERVATION"
    error_message = "a flex-start pool must not consume reservations"
  }
}

run "spot_pools_render_alike_from_either_spelling" {
  command = plan

  assert {
    condition     = google_container_node_pool.pools["spot"].node_config[0].spot == true
    error_message = "provisioning_model = \"spot\" must render a spot pool"
  }

  assert {
    condition     = google_container_node_pool.pools["spot-alias"].node_config[0].spot == true
    error_message = "spot = true must still render a spot pool"
  }

  assert {
    condition     = google_container_node_pool.pools["spot-alias"].node_config[0].flex_start == false
    error_message = "a spot pool must not be flex-start"
  }

  assert {
    condition     = google_container_node_pool.pools["spot-alias"].management[0].auto_repair == true
    error_message = "a spot pool must keep auto-repair on"
  }
}

run "plain_pool_keeps_the_defaults" {
  command = plan

  assert {
    condition     = google_container_node_pool.pools["plain"].node_config[0].spot == false
    error_message = "a pool with no provisioning model must be on-demand"
  }

  assert {
    condition     = google_container_node_pool.pools["plain"].node_config[0].flex_start == false
    error_message = "a pool with no provisioning model must not be flex-start"
  }

  assert {
    condition     = google_container_node_pool.pools["plain"].management[0].auto_repair == true
    error_message = "a pool with no provisioning model must keep auto-repair on"
  }

  assert {
    condition     = length(google_container_node_pool.pools["plain"].node_config[0].reservation_affinity) == 0
    error_message = "a pool with no provisioning model must not set reservation affinity"
  }
}

run "unknown_provisioning_model_is_refused" {
  command = plan

  variables {
    pools = {
      bad = {
        provisioning_model = "preemptible"
      }
    }
  }

  expect_failures = [var.pools]
}

run "spot_alias_disagreeing_with_the_model_is_refused" {
  command = plan

  variables {
    pools = {
      both = {
        spot               = true
        provisioning_model = "flex-start"
      }
    }
  }

  expect_failures = [var.pools]
}

run "max_run_duration_without_flex_start_is_refused" {
  command = plan

  variables {
    pools = {
      timed = {
        provisioning_model = "on-demand"
        max_run_duration   = "3600s"
      }
    }
  }

  expect_failures = [var.pools]
}
