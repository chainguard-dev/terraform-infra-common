# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0
#
# networking: how each region's Cloud Run subnet is numbered. Run from
# modules/networking with
#   terraform init -backend=false && terraform test

mock_provider "google" {}
mock_provider "random" {}

variables {
  name       = "fixture"
  project_id = "fixture-project"
  team       = "fixture"
}

run "numbers_regions_by_position" {
  command = plan

  variables {
    regions       = ["us-central1", "us-east4"]
    netnum_offset = 20
  }

  assert {
    condition = (
      google_compute_subnetwork.regional["us-central1"].ip_cidr_range == "10.20.0.0/16" &&
      google_compute_subnetwork.regional["us-east4"].ip_cidr_range == "10.21.0.0/16"
    )
    error_message = "Without region_netnums, each region's /16 is netnum_offset plus its position in regions."
  }
}

run "region_netnums_skip_ranges" {
  command = plan

  variables {
    regions        = ["us-central1", "us-east4", "us-west1"]
    region_netnums = { "us-east4" = 2, "us-west1" = 3 }
  }

  assert {
    condition = (
      google_compute_subnetwork.regional["us-central1"].ip_cidr_range == "10.0.0.0/16" &&
      google_compute_subnetwork.regional["us-east4"].ip_cidr_range == "10.2.0.0/16" &&
      google_compute_subnetwork.regional["us-west1"].ip_cidr_range == "10.3.0.0/16"
    )
    error_message = "A region in region_netnums takes that netnum; the others keep their positions."
  }
}

run "refuses_a_shared_netnum" {
  command = plan

  variables {
    regions        = ["us-central1", "us-east4"]
    region_netnums = { "us-east4" = 0 }
  }

  expect_failures = [var.region_netnums]
}

run "refuses_an_unknown_region" {
  command = plan

  variables {
    regions        = ["us-central1"]
    region_netnums = { "us-east4" = 2 }
  }

  expect_failures = [var.region_netnums]
}
