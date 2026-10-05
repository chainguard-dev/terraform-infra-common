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
    bluegreen = {
      upgrade_settings = {
        strategy = "BLUE_GREEN"
        blue_green_settings = {
          node_pool_soak_duration = "43200s"
          standard_rollout_policy = {
            batch_node_count    = 1
            batch_soak_duration = "0s"
          }
        }
      }
    }
    surge = {
      upgrade_settings = {
        max_surge       = 2
        max_unavailable = 0
      }
    }
  }
}

run "blue_green_pool_renders_the_strategy_and_soak" {
  command = plan

  assert {
    condition     = google_container_node_pool.pools["bluegreen"].upgrade_settings[0].strategy == "BLUE_GREEN"
    error_message = "a blue-green pool must set upgrade_settings.strategy = BLUE_GREEN"
  }

  assert {
    condition     = google_container_node_pool.pools["bluegreen"].upgrade_settings[0].blue_green_settings[0].node_pool_soak_duration == "43200s"
    error_message = "a blue-green pool must carry node_pool_soak_duration through unchanged"
  }

  assert {
    condition     = google_container_node_pool.pools["bluegreen"].upgrade_settings[0].blue_green_settings[0].standard_rollout_policy[0].batch_node_count == 1
    error_message = "a blue-green pool must carry standard_rollout_policy.batch_node_count through unchanged"
  }

  assert {
    condition     = google_container_node_pool.pools["surge"].upgrade_settings[0].strategy == "SURGE"
    error_message = "a pool with surge counts and no strategy must default to SURGE"
  }

  assert {
    condition     = google_container_node_pool.pools["surge"].upgrade_settings[0].max_surge == 2
    error_message = "a surge pool must carry max_surge through unchanged"
  }
}

run "blue_green_without_settings_is_refused" {
  command = plan

  variables {
    pools = {
      broken = {
        upgrade_settings = { strategy = "BLUE_GREEN" }
      }
    }
  }

  expect_failures = [var.pools]
}

run "blue_green_with_surge_counts_is_refused" {
  command = plan

  variables {
    pools = {
      broken = {
        upgrade_settings = {
          strategy  = "BLUE_GREEN"
          max_surge = 1
          blue_green_settings = {
            standard_rollout_policy = { batch_node_count = 1 }
          }
        }
      }
    }
  }

  expect_failures = [var.pools]
}

run "surge_without_counts_is_refused" {
  command = plan

  variables {
    pools = {
      broken = {
        upgrade_settings = { strategy = "SURGE" }
      }
    }
  }

  expect_failures = [var.pools]
}

run "rollout_policy_needs_exactly_one_batch_size" {
  command = plan

  variables {
    pools = {
      broken = {
        upgrade_settings = {
          strategy = "BLUE_GREEN"
          blue_green_settings = {
            standard_rollout_policy = {
              batch_node_count = 1
              batch_percentage = 0.5
            }
          }
        }
      }
    }
  }

  expect_failures = [var.pools]
}
