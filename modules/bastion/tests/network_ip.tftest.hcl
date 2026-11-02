# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0
#
# Run in CI by .github/workflows/tf-module-tests.yaml.

mock_provider "google" {}

variables {
  name           = "fixture"
  project_id     = "fixture-project"
  zone           = "us-central1-a"
  network        = "fixture-network"
  subnetwork     = "fixture-subnetwork"
  dev_principals = []
  team           = "fixture"
}

run "network_ip_defaults_to_ephemeral" {
  command = plan
  assert {
    condition     = var.network_ip == null
    error_message = "network_ip must default to null so existing callers keep an ephemeral address"
  }
}

run "network_ip_sets_the_interface_address" {
  command = plan
  variables {
    network_ip = "10.0.0.10"
  }
  assert {
    condition     = google_compute_instance.bastion.network_interface[0].network_ip == "10.0.0.10"
    error_message = "network_ip must reach the instance's network interface"
  }
}
