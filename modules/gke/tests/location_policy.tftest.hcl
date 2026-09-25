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
  # A pool that leaves location_policy unset sends none, so GKE keeps its
  # default (BALANCED); the provider computes the field, so a plan can't
  # assert that here, and callers' own plans show those pools unchanged.
  pools = {
    any = {
      min_node_count  = 0
      max_node_count  = 5
      location_policy = "ANY"
    }
    flex = {
      min_node_count     = 0
      max_node_count     = 5
      provisioning_model = "flex-start"
      max_run_duration   = "57600s"
    }
  }
}

run "location_policy_rides_through" {
  command = plan

  assert {
    condition     = google_container_node_pool.pools["any"].autoscaling[0].location_policy == "ANY"
    error_message = "location_policy must reach the pool's autoscaling block"
  }

  # GKE requires ANY on flex-start pools whatever the pool sets.
  assert {
    condition     = google_container_node_pool.pools["flex"].autoscaling[0].location_policy == "ANY"
    error_message = "a flex-start pool must keep location policy ANY"
  }
}

run "rejects_unknown_location_policy" {
  command = plan

  variables {
    pools = {
      bad = {
        min_node_count  = 0
        max_node_count  = 5
        location_policy = "SPREAD"
      }
    }
  }

  expect_failures = [var.pools]
}
