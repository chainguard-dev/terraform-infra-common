# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0
#
# Run in CI by .github/workflows/tf-module-tests.yaml.
#
# Plan-only tests that pin the rendered Cloud Run Job vpc_access block, mirroring
# regional-service/tests/vpc-access.tftest.hcl for the job (not service) resource.
#
# Cloud Run accepts a Serverless VPC Access connector OR direct VPC egress
# (network_interfaces), never both, so regional-connector has to suppress the
# network_interfaces block for the regions it covers. Getting this wrong fails
# silently: a region left on direct VPC egress when the caller meant to move it
# onto a connector just keeps allocating a network interface per execution.
#
# Mock providers keep this fully offline: no credentials, no state.

mock_provider "google" {
  mock_data "google_project" {
    defaults = {
      number = "123456789"
    }
  }
}
mock_provider "google-beta" {}
mock_provider "ko" {}
mock_provider "cosign" {}

variables {
  project_id      = "fixture-project"
  name            = "fixture"
  service_account = "fixture@fixture-project.iam.gserviceaccount.com"
  team            = "fixture"
  regions = {
    "us-central1" = {
      network = "projects/fixture-project/global/networks/fixture"
      subnet  = "projects/fixture-project/regions/us-central1/subnetworks/fixture"
    }
    "us-west1" = {
      network = "projects/fixture-project/global/networks/fixture"
      subnet  = "projects/fixture-project/regions/us-west1/subnetworks/fixture"
    }
  }
  regional-cronspec = {
    "us-central1" = { schedule = "0 * * * *" }
    "us-west1"    = { schedule = "0 * * * *" }
  }
}

# With regional-connector unset, every region keeps direct VPC egress exactly
# as before and no connector is attached.
run "no_connector_renders_direct_vpc_egress" {
  command = plan

  assert {
    condition     = length(google_cloud_run_v2_job.this["us-central1"].template[0].template[0].vpc_access[0].network_interfaces) == 1
    error_message = "region without a connector must keep its network_interfaces block (direct VPC egress)"
  }

  assert {
    condition     = google_cloud_run_v2_job.this["us-central1"].template[0].template[0].vpc_access[0].connector == null
    error_message = "region without a connector must not render a connector"
  }
}

# A region present in regional-connector attaches the connector AND drops
# network_interfaces, which Cloud Run would otherwise reject as conflicting.
run "connector_replaces_direct_vpc_egress" {
  command = plan

  variables {
    regional-connector = {
      "us-central1" = "projects/host-project/locations/us-central1/connectors/cr-egress-us-central1"
    }
  }

  assert {
    condition     = google_cloud_run_v2_job.this["us-central1"].template[0].template[0].vpc_access[0].connector == "projects/host-project/locations/us-central1/connectors/cr-egress-us-central1"
    error_message = "region with a connector must render that connector"
  }

  assert {
    condition     = length(google_cloud_run_v2_job.this["us-central1"].template[0].template[0].vpc_access[0].network_interfaces) == 0
    error_message = "region with a connector must NOT also render network_interfaces; Cloud Run takes one or the other"
  }
}

# The map is per-region: covering one region must not disturb the others.
run "connector_is_scoped_to_its_region" {
  command = plan

  variables {
    regional-connector = {
      "us-central1" = "projects/host-project/locations/us-central1/connectors/cr-egress-us-central1"
    }
  }

  assert {
    condition     = google_cloud_run_v2_job.this["us-west1"].template[0].template[0].vpc_access[0].connector == null
    error_message = "a region absent from regional-connector must not inherit another region's connector"
  }

  assert {
    condition     = length(google_cloud_run_v2_job.this["us-west1"].template[0].template[0].vpc_access[0].network_interfaces) == 1
    error_message = "a region absent from regional-connector must keep direct VPC egress"
  }
}

# A bare connector name resolves in the job's own project, which silently is
# not where a Shared VPC host-project connector lives. Reject it at plan time.
run "bare_connector_name_is_rejected" {
  command = plan

  variables {
    regional-connector = {
      "us-central1" = "cr-egress-us-central1"
    }
  }

  expect_failures = [
    var.regional-connector,
  ]
}
