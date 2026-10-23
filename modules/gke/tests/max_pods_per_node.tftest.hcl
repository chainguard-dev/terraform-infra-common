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
    dense = {
      max_pods_per_node = 32
    }
    gke-maximum = {
      max_pods_per_node = 512
    }
  }
}

run "set_value_passes_through" {
  command = plan

  assert {
    condition     = google_container_node_pool.pools["dense"].max_pods_per_node == 32
    error_message = "max_pods_per_node must carry through to the node pool unchanged"
  }

  assert {
    condition     = google_container_node_pool.pools["gke-maximum"].max_pods_per_node == 512
    error_message = "max_pods_per_node must accept 512, GKE's Standard maximum"
  }
}

# The provider computes max_pods_per_node, so an unset one is unknown on
# the planned resource and cannot be asserted there. The module's part is
# passing null, which leaves the cluster default.
run "unset_value_stays_null" {
  command = plan

  assert {
    condition     = var.pools["plain"].max_pods_per_node == null
    error_message = "max_pods_per_node must default to null so existing pools keep the cluster default"
  }
}

run "below_gke_minimum_is_rejected" {
  command = plan

  variables {
    pools = {
      wrong = {
        max_pods_per_node = 4
      }
    }
  }

  expect_failures = [var.pools]
}

run "above_gke_maximum_is_rejected" {
  command = plan

  variables {
    pools = {
      wrong = {
        max_pods_per_node = 513
      }
    }
  }

  expect_failures = [var.pools]
}
