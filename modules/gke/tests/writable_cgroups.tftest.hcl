# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0
mock_provider "google" {
  mock_resource "google_container_cluster" {
    override_during = apply
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
    plain     = {}
    delegated = { writable_cgroups = true }
  }
}
run "delegation_opt_in" {
  command = plan
  assert {
    condition     = google_container_node_pool.pools["delegated"].node_config[0].containerd_config[0].writable_cgroups[0].enabled
    error_message = "delegated pool must enable containerd writable cgroups"
  }
  assert {
    condition     = length(google_container_node_pool.pools["plain"].node_config[0].containerd_config) == 0
    error_message = "plain pool must omit containerd configuration"
  }
}

run "existing_pools" {
  command = apply
  module {
    source = "./tests/lifecycle"
  }
  variables {
    pools = { plain = {}, delegated = {} }
  }
}

run "enable_replaces_only_selected_pool" {
  command = apply
  module {
    source = "./tests/lifecycle"
  }
  assert {
    condition     = google_container_node_pool.pools["plain"].id == terraform_data.original_pool_ids["plain"].output
    error_message = "Enabling delegated must retain the existing plain node pool."
  }
  assert {
    condition     = google_container_node_pool.pools["delegated"].id != terraform_data.original_pool_ids["delegated"].output
    error_message = "Enabling writable cgroups must replace the existing delegated pool, not issue an unsupported in-place update."
  }
}
