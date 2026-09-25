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
    per-zone = {
      min_node_count = 0
      max_node_count = 5
    }
    total = {
      node_locations       = ["us-central1-a", "us-central1-b", "us-central1-c"]
      total_min_node_count = 1
      total_max_node_count = 15
    }
  }
}

run "per_zone_bounds_stay_the_default" {
  command = plan

  assert {
    condition = (
      google_container_node_pool.pools["per-zone"].autoscaling[0].min_node_count == 0 &&
      google_container_node_pool.pools["per-zone"].autoscaling[0].max_node_count == 5 &&
      google_container_node_pool.pools["per-zone"].autoscaling[0].total_min_node_count == null &&
      google_container_node_pool.pools["per-zone"].autoscaling[0].total_max_node_count == null
    )
    error_message = "a pool that sets no totals must keep its per-zone bounds and set no total ones"
  }
}

run "total_bounds_replace_the_per_zone_ones" {
  command = plan

  assert {
    condition = (
      google_container_node_pool.pools["total"].autoscaling[0].total_min_node_count == 1 &&
      google_container_node_pool.pools["total"].autoscaling[0].total_max_node_count == 15
    )
    error_message = "total_min_node_count / total_max_node_count must reach the pool's autoscaling block"
  }

  # GKE refuses per-zone and total bounds together, and the per-zone
  # defaults (1/1) would otherwise ride along.
  assert {
    condition = (
      google_container_node_pool.pools["total"].autoscaling[0].min_node_count == null &&
      google_container_node_pool.pools["total"].autoscaling[0].max_node_count == null
    )
    error_message = "a pool with total bounds must send no per-zone bounds"
  }
}

run "totals_must_be_set_together" {
  command = plan

  variables {
    pools = {
      half = {
        total_min_node_count = 1
      }
    }
  }

  expect_failures = [var.pools]
}

run "total_min_must_not_exceed_total_max" {
  command = plan

  variables {
    pools = {
      inverted = {
        total_min_node_count = 3
        total_max_node_count = 2
      }
    }
  }

  expect_failures = [var.pools]
}
