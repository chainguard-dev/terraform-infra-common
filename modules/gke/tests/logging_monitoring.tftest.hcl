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
  pools      = {}
}

run "no_logging_config_by_default" {
  command = plan

  assert {
    condition     = length(google_container_cluster.this.logging_config) == 0
    error_message = "an unset logging_components must render no logging_config block"
  }

  assert {
    condition     = google_container_cluster.this.monitoring_config[0].enable_components == toset(["SYSTEM_COMPONENTS", "APISERVER", "SCHEDULER", "CONTROLLER_MANAGER", "STORAGE", "POD"])
    error_message = "the monitoring_components default must keep the components the module has always enabled"
  }

  assert {
    condition     = output.logging_components == null
    error_message = "the logging_components output must be null when no logging_config block renders"
  }
}

run "components_ride_through" {
  command = plan

  variables {
    logging_components    = ["SYSTEM_COMPONENTS", "WORKLOADS"]
    monitoring_components = ["SYSTEM_COMPONENTS", "POD"]
  }

  assert {
    condition     = google_container_cluster.this.logging_config[0].enable_components == toset(["SYSTEM_COMPONENTS", "WORKLOADS"])
    error_message = "logging_config.enable_components must carry logging_components through unchanged"
  }

  assert {
    condition     = google_container_cluster.this.monitoring_config[0].enable_components == toset(["SYSTEM_COMPONENTS", "POD"])
    error_message = "monitoring_config.enable_components must carry monitoring_components through unchanged"
  }

  assert {
    condition     = output.logging_components == toset(["SYSTEM_COMPONENTS", "WORKLOADS"]) && output.monitoring_components == toset(["SYSTEM_COMPONENTS", "POD"])
    error_message = "the logging_components and monitoring_components outputs must report the rendered components, so a wrapper module's tests can assert on them"
  }
}

run "rejects_gcloud_component_names" {
  command = plan

  variables {
    logging_components = ["SYSTEM", "WORKLOAD"]
  }

  expect_failures = [var.logging_components]
}

run "null_monitoring_components_take_the_default" {
  command = plan

  variables {
    monitoring_components = null
  }

  assert {
    condition     = google_container_cluster.this.monitoring_config[0].enable_components == toset(["SYSTEM_COMPONENTS", "APISERVER", "SCHEDULER", "CONTROLLER_MANAGER", "STORAGE", "POD"])
    error_message = "a null monitoring_components, as a wrapper module passes when unset, must keep the default components"
  }
}
