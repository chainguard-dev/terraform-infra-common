# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

mock_provider "google" {}

variables {
  project                = "fixture-project"
  region                 = "us-central1"
  name                   = "fixture"
  network                = "projects/fixture-project/global/networks/fixture"
  subnetwork             = "projects/fixture-project/regions/us-central1/subnetworks/frontend"
  proxy_only_subnet      = "projects/fixture-project/regions/us-central1/subnetworks/proxy"
  psc_nat_subnets        = ["projects/fixture-project/regions/us-central1/subnetworks/psc-nat"]
  cloud_run_service_name = "fixture"
}

run "projects_accept_every_network_in_the_project" {
  command = plan

  variables {
    consumer_accept_projects = ["consumer-project"]
  }

  assert {
    condition     = one(google_compute_service_attachment.this.consumer_accept_lists).project_id_or_num == "consumer-project"
    error_message = "A project entry must render as project_id_or_num."
  }
}

# Accepting a network instead of its project keeps endpoints that other
# networks in the same project create (PENDING) off the service.
run "networks_accept_only_the_named_network" {
  command = plan

  variables {
    consumer_accept_networks = ["https://www.googleapis.com/compute/v1/projects/consumer-project/global/networks/access"]
  }

  assert {
    condition = (
      one(google_compute_service_attachment.this.consumer_accept_lists).network_url == "https://www.googleapis.com/compute/v1/projects/consumer-project/global/networks/access" &&
      one(google_compute_service_attachment.this.consumer_accept_lists).project_id_or_num == null &&
      one(google_compute_service_attachment.this.consumer_accept_lists).connection_limit == 10
    )
    error_message = "A network entry must render as network_url alone, with the shared connection limit."
  }
}

run "relative_network_paths_are_accepted" {
  command = plan

  variables {
    consumer_accept_networks = ["projects/consumer-project/global/networks/access"]
  }

  assert {
    condition     = one(google_compute_service_attachment.this.consumer_accept_lists).network_url == "projects/consumer-project/global/networks/access"
    error_message = "A relative network path must pass through unchanged."
  }
}

run "mixed_consumer_types_rejected" {
  command = plan

  variables {
    consumer_accept_projects = ["consumer-project"]
    consumer_accept_networks = ["projects/consumer-project/global/networks/access"]
  }

  expect_failures = [var.consumer_accept_networks]
}

run "project_id_in_network_list_rejected" {
  command = plan

  variables {
    consumer_accept_networks = ["consumer-project"]
  }

  expect_failures = [var.consumer_accept_networks]
}
