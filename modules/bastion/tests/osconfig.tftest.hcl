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

run "enable_osconfig_defaults_to_unchanged_metadata" {
  command = plan
  assert {
    condition = google_compute_instance.bastion.metadata == tomap({
      enable-oslogin         = "TRUE"
      block-project-ssh-keys = "TRUE"
      enable-oslogin-2fa     = "TRUE"
    })
    error_message = "With enable_osconfig unset the metadata must be unchanged, so existing bastions plan no change"
  }
}

run "enable_osconfig_sets_the_metadata_key" {
  command = plan
  variables {
    enable_osconfig = true
  }
  assert {
    condition = google_compute_instance.bastion.metadata == tomap({
      enable-oslogin         = "TRUE"
      block-project-ssh-keys = "TRUE"
      enable-oslogin-2fa     = "TRUE"
      enable-osconfig        = "TRUE"
    })
    error_message = "enable_osconfig adds enable-osconfig = TRUE and keeps the OS Login keys"
  }
}
