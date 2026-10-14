# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0
#
# A caller that sets no product gets no product label, so the service inherits
# its owner from its project. A caller that sets one keeps it.

mock_provider "google-beta" {}

mock_provider "google" {
  mock_data "google_project" {
    defaults = {
      number = "123456789"
    }
  }
}

variables {
  project_id = "fixture-project"
  name       = "fixture"
  regions = {
    "us-central1" = {
      network = "projects/fixture-project/global/networks/fixture"
      subnet  = "projects/fixture-project/regions/us-central1/subnetworks/fixture"
    }
  }
  service_account       = "fixture@fixture-project.iam.gserviceaccount.com"
  notification_channels = []
  team                  = "fixture"
  containers = {
    "main" = {
      image = "cgr.dev/chainguard/static:latest"
      ports = [{ container_port = 8080 }]
    }
  }
}

run "unset_product_sets_no_product_label" {
  command = plan

  assert {
    condition     = !contains(keys(google_cloud_run_v2_service.this["us-central1"].labels), "product")
    error_message = "A caller that sets no product must get no product label."
  }
}

run "set_product_keeps_its_label" {
  command = plan

  variables {
    product = "fixture-product"
  }

  assert {
    condition     = google_cloud_run_v2_service.this["us-central1"].labels["product"] == "fixture-product"
    error_message = "A caller's product must reach the service's product label."
  }
}
