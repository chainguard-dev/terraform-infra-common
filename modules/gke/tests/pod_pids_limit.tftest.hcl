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
    bounded = {
      pod_pids_limit = 4096
    }
  }
}

run "set_value_renders_the_kubelet_config" {
  command = plan

  assert {
    condition     = google_container_node_pool.pools["bounded"].node_config[0].kubelet_config[0].pod_pids_limit == 4096
    error_message = "pod_pids_limit must carry through to the pool's kubelet_config unchanged"
  }

  assert {
    condition     = google_container_node_pool.pools["bounded"].node_config[0].kubelet_config[0].cpu_cfs_quota == true
    error_message = "a kubelet_config emitted for the pids limit must keep CPU CFS quota on, GKE's default"
  }
}

# A pool that sets no limit emits no kubelet_config block at all, so existing
# callers plan no change; the module's part is passing null through.
run "unset_value_emits_no_block" {
  command = plan

  assert {
    condition     = var.pools["plain"].pod_pids_limit == null
    error_message = "pod_pids_limit must default to null"
  }

  assert {
    condition     = length(google_container_node_pool.pools["plain"].node_config[0].kubelet_config) == 0
    error_message = "a pool without pod_pids_limit must emit no kubelet_config block"
  }
}

run "below_gke_minimum_is_rejected" {
  command = plan

  variables {
    pools = {
      wrong = {
        pod_pids_limit = 512
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
        pod_pids_limit = 4194305
      }
    }
  }

  expect_failures = [var.pools]
}
