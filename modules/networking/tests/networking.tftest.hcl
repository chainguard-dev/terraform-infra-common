# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0
#
# networking: subnet numbering and the optional run.app private zone. Run from
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

run "resolves_run_app_privately_by_default" {
  command = plan

  variables {
    regions = ["us-central1"]
  }

  assert {
    condition = (
      length(google_dns_managed_zone.cloud-run-internal) == 1 &&
      google_dns_managed_zone.cloud-run-internal[0].dns_name == "run.app." &&
      length(google_dns_record_set.cloud-run-cname) == 1 &&
      tolist(google_dns_record_set.cloud-run-cname[0].rrdatas) == tolist(["private.googleapis.com."])
    )
    error_message = "By default *.run.app resolves to private.googleapis.com."
  }
}

run "run_app_private_dns_disabled" {
  command = plan

  variables {
    regions                     = ["us-central1"]
    run_app_private_dns_enabled = false
  }

  assert {
    condition = (
      length(google_dns_managed_zone.cloud-run-internal) == 0 &&
      length(google_dns_record_set.cloud-run-cname) == 0
    )
    error_message = "With run_app_private_dns_enabled = false, no run.app zone or record is created."
  }

  assert {
    condition = (
      google_dns_managed_zone.private-google-apis.dns_name == "private.googleapis.com." &&
      length(google_dns_record_set.private-googleapis-a-record.rrdatas) == 4
    )
    error_message = "The private.googleapis.com zone stays when run.app private DNS is off."
  }
}
