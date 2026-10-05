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
  }
}

run "unset_emits_no_timeouts_block" {
  command = plan

  assert {
    condition     = google_container_node_pool.pools["plain"].timeouts == null
    error_message = "with node_pool_timeouts unset the pool must carry no timeouts block, so existing callers plan no change"
  }
}

run "set_renders_only_the_given_timeouts" {
  command = plan

  variables {
    node_pool_timeouts = {
      update = "4h"
      delete = "4h"
    }
  }

  assert {
    condition     = google_container_node_pool.pools["plain"].timeouts.update == "4h" && google_container_node_pool.pools["plain"].timeouts.delete == "4h"
    error_message = "update and delete must ride through unchanged"
  }

  assert {
    condition     = google_container_node_pool.pools["plain"].timeouts.create == null
    error_message = "an attribute left unset must stay null so the provider default applies"
  }
}
