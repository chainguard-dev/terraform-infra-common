# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

mock_provider "google" {
  mock_resource "google_container_cluster" {
    override_during = plan
    defaults = {
      endpoint = "203.0.113.2"
      private_cluster_config = {
        private_endpoint = "10.0.0.2"
      }
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
    primary = {}
  }
}

run "distinct_cluster_endpoints" {
  command = plan
  assert {
    condition     = output.cluster_private_endpoint == "10.0.0.2"
    error_message = "The private endpoint output must select the private control-plane IP."
  }
  assert {
    condition     = output.cluster_endpoint == "203.0.113.2"
    error_message = "The existing endpoint output must retain the cluster endpoint."
  }
}
