# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

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
    "us-east4" = {
      network = "projects/fixture-project/global/networks/fixture"
      subnet  = "projects/fixture-project/regions/us-east4/subnetworks/fixture"
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

# Cloud Run counts every same-project VM, pod, and Google service as internal
# on the run.app URL, so a service meant to answer only through its internal
# load balancer must turn that URL off in every region it runs in.
run "disables_default_uri_in_every_region" {
  command = plan

  variables {
    default_uri_disabled = true
  }

  assert {
    condition     = length(google_cloud_run_v2_service.this) == 2 && alltrue([for s in google_cloud_run_v2_service.this : s.default_uri_disabled])
    error_message = "default_uri_disabled must reach the service in every region."
  }
}
