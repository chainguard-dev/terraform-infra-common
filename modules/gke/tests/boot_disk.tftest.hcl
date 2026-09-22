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
    provisioned = {
      disk_type                        = "hyperdisk-balanced"
      disk_size                        = 2048
      boot_disk_provisioned_iops       = 6000
      boot_disk_provisioned_throughput = 1200
    }
    throughput-only = {
      disk_type                        = "hyperdisk-balanced"
      boot_disk_provisioned_throughput = 500
    }
  }
}

run "provisioned_pool_renders_the_boot_disk_block" {
  command = plan

  assert {
    condition     = length(google_container_node_pool.pools["provisioned"].node_config[0].boot_disk) == 1
    error_message = "a pool with provisioned boot-disk performance must render one boot_disk block"
  }

  assert {
    condition     = google_container_node_pool.pools["provisioned"].node_config[0].boot_disk[0].provisioned_iops == 6000
    error_message = "boot_disk.provisioned_iops must carry boot_disk_provisioned_iops through unchanged"
  }

  assert {
    condition     = google_container_node_pool.pools["provisioned"].node_config[0].boot_disk[0].provisioned_throughput == 1200
    error_message = "boot_disk.provisioned_throughput must carry boot_disk_provisioned_throughput through unchanged"
  }

  # The provider requires the legacy and boot_disk spellings of type and
  # size to agree while both are set.
  assert {
    condition     = google_container_node_pool.pools["provisioned"].node_config[0].boot_disk[0].disk_type == google_container_node_pool.pools["provisioned"].node_config[0].disk_type
    error_message = "boot_disk.disk_type must repeat node_config.disk_type"
  }

  assert {
    condition     = google_container_node_pool.pools["provisioned"].node_config[0].boot_disk[0].size_gb == google_container_node_pool.pools["provisioned"].node_config[0].disk_size_gb
    error_message = "boot_disk.size_gb must repeat node_config.disk_size_gb"
  }
}

run "either_field_alone_renders_the_block" {
  command = plan

  assert {
    condition     = length(google_container_node_pool.pools["throughput-only"].node_config[0].boot_disk) == 1
    error_message = "provisioning throughput alone must still render the boot_disk block"
  }

  # provisioned_iops is Computed, so an unset one is unknown at plan time
  # (the API reports the baseline back) and cannot be asserted here.
  assert {
    condition     = google_container_node_pool.pools["throughput-only"].node_config[0].boot_disk[0].provisioned_throughput == 500
    error_message = "boot_disk.provisioned_throughput must carry through when it is the only provisioned field"
  }
}

run "unprovisioned_pool_keeps_its_plan" {
  command = plan

  assert {
    condition     = length(google_container_node_pool.pools["plain"].node_config[0].boot_disk) == 0
    error_message = "a pool without provisioned performance must not render a boot_disk block"
  }
}

run "provisioned_performance_requires_hyperdisk_balanced" {
  command = plan

  variables {
    pools = {
      wrong = {
        disk_type                        = "pd-balanced"
        boot_disk_provisioned_throughput = 500
      }
    }
  }

  expect_failures = [var.pools]
}
