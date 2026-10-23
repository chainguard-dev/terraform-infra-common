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
  name                       = "fixture"
  project                    = "fixture-project"
  network                    = "fixture-network"
  region                     = "us-central1"
  team                       = "fixture"
  subnetwork                 = "fixture-subnetwork"
  node_resource_manager_tags = { "tagKeys/123" = "tagValues/456" }
}

run "pools_inherit_module_tags_and_team_by_default" {
  command = plan

  variables {
    pools = { p = {} }
  }

  assert {
    condition     = google_container_node_pool.pools["p"].node_config[0].resource_manager_tags["tagKeys/123"] == "tagValues/456"
    error_message = "a pool with no override must carry the module-wide node tags"
  }

  assert {
    condition     = google_container_node_pool.pools["p"].node_config[0].resource_labels["team"] == "fixture" && google_container_node_pool.pools["p"].node_config[0].resource_labels["squad"] == "fixture"
    error_message = "a pool with no override must carry var.team as its squad and team labels"
  }
}

run "empty_pool_tags_clear_only_that_pool" {
  command = plan

  variables {
    pools = {
      held = { node_resource_manager_tags = {} }
      p    = {}
    }
  }

  assert {
    condition     = google_container_node_pool.pools["held"].node_config[0].resource_manager_tags == null
    error_message = "node_resource_manager_tags = {} must leave the pool with no node tags"
  }

  assert {
    condition     = google_container_node_pool.pools["p"].node_config[0].resource_manager_tags["tagKeys/123"] == "tagValues/456"
    error_message = "other pools must keep the module-wide node tags"
  }
}

run "pool_tags_replace_module_tags" {
  command = plan

  variables {
    pools = {
      p = { node_resource_manager_tags = { "tagKeys/789" = "tagValues/12" } }
    }
  }

  assert {
    condition     = google_container_node_pool.pools["p"].node_config[0].resource_manager_tags == tomap({ "tagKeys/789" = "tagValues/12" })
    error_message = "a pool's own node tags must replace the module-wide map"
  }
}

run "team_override_changes_only_that_pool" {
  command = plan

  variables {
    pools = {
      held = { team = "other" }
      p    = {}
    }
  }

  assert {
    condition     = google_container_node_pool.pools["held"].node_config[0].resource_labels["team"] == "other" && google_container_node_pool.pools["held"].node_config[0].resource_labels["squad"] == "other"
    error_message = "a pool's team must set its squad and team labels"
  }

  assert {
    condition     = google_container_node_pool.pools["p"].node_config[0].resource_labels["team"] == "fixture" && google_container_node_pool.pools["p"].node_config[0].resource_labels["squad"] == "fixture"
    error_message = "other pools must keep var.team"
  }
}

run "refuses_malformed_pool_tags" {
  command = plan

  variables {
    pools = {
      p = { node_resource_manager_tags = { "owner" = "me" } }
    }
  }

  expect_failures = [var.pools]
}
