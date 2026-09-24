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
    no-repair = {
      auto_repair = false
    }
    flex = {
      provisioning_model = "flex-start"
      max_run_duration   = "57600s"
    }
  }
}

run "auto_repair_defaults_on_and_is_settable" {
  command = plan

  assert {
    condition     = google_container_node_pool.pools["plain"].management[0].auto_repair == true
    error_message = "a pool that sets nothing must keep auto-repair on"
  }

  assert {
    condition     = google_container_node_pool.pools["no-repair"].management[0].auto_repair == false
    error_message = "auto_repair = false must turn auto-repair off for that pool"
  }

  assert {
    condition     = google_container_node_pool.pools["flex"].management[0].auto_repair == false
    error_message = "a flex-start pool must keep auto-repair off, as GKE requires"
  }

  assert {
    condition     = google_container_node_pool.pools["no-repair"].management[0].auto_upgrade == true
    error_message = "turning auto-repair off must leave auto-upgrade on"
  }
}
