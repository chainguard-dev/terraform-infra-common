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
    ubuntu = {
      image_type = "UBUNTU_CONTAINERD"
    }
  }
}

# A pool that says nothing keeps Container-Optimized OS, the image every
# caller had before the attribute existed, so existing pools plan no change.
run "unset_value_keeps_cos" {
  command = plan

  assert {
    condition     = var.pools["plain"].image_type == "COS_CONTAINERD"
    error_message = "image_type must default to COS_CONTAINERD"
  }

  assert {
    condition     = google_container_node_pool.pools["plain"].node_config[0].image_type == "COS_CONTAINERD"
    error_message = "a pool without image_type must render COS_CONTAINERD"
  }
}

run "set_value_renders_the_image_type" {
  command = plan

  assert {
    condition     = google_container_node_pool.pools["ubuntu"].node_config[0].image_type == "UBUNTU_CONTAINERD"
    error_message = "image_type must carry through to the pool's node_config unchanged"
  }
}

run "unknown_image_is_rejected" {
  command = plan

  variables {
    pools = {
      wrong = {
        image_type = "COS"
      }
    }
  }

  expect_failures = [var.pools]
}

# GKE Sandbox needs Container-Optimized OS, so the two settings cannot meet.
run "gvisor_on_ubuntu_is_rejected" {
  command = plan

  variables {
    pools = {
      wrong = {
        image_type = "UBUNTU_CONTAINERD"
        gvisor     = true
      }
    }
  }

  expect_failures = [var.pools]
}
