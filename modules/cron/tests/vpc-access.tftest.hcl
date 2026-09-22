# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0
#
# Run in CI by .github/workflows/tf-module-tests.yaml.
#
# This module has no resource of its own to inspect — it wraps regional-go-cron
# for a single region (see main.tf's module "impl"), which regional-go-cron's
# own tests/vpc-access.tftest.hcl covers at the resource level. What's testable
# here is this module's own contract: vpc_access must set exactly one of
# network_interfaces or connector, and a connector-only value must still plan
# cleanly through the wrapper (network_interfaces defaults to empty).
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
  project_id            = "fixture-project"
  name                  = "fixture"
  schedule              = "0 * * * *"
  service_account       = "fixture@fixture-project.iam.gserviceaccount.com"
  working_dir           = "."
  importpath            = "example.com/fixture/cmd/app"
  team                  = "fixture"
  notification_channels = []
}

# The pre-existing shape (network_interfaces, no connector) must still plan.
run "network_interfaces_only_plans_cleanly" {
  command = plan

  variables {
    vpc_access = {
      network_interfaces = [{
        network    = "projects/fixture-project/global/networks/fixture"
        subnetwork = "projects/fixture-project/regions/us-east4/subnetworks/fixture"
      }]
      egress = "PRIVATE_RANGES_ONLY"
    }
  }
}

# A connector-only value (no network_interfaces) must also plan cleanly.
run "connector_only_plans_cleanly" {
  command = plan

  variables {
    vpc_access = {
      connector = "projects/host-project/locations/us-east4/connectors/cr-egress-us-east4"
      egress    = "PRIVATE_RANGES_ONLY"
    }
  }
}

# Setting both is rejected: Cloud Run accepts one or the other, never both.
run "both_network_interfaces_and_connector_is_rejected" {
  command = plan

  variables {
    vpc_access = {
      network_interfaces = [{
        network    = "projects/fixture-project/global/networks/fixture"
        subnetwork = "projects/fixture-project/regions/us-east4/subnetworks/fixture"
      }]
      connector = "projects/host-project/locations/us-east4/connectors/cr-egress-us-east4"
      egress    = "PRIVATE_RANGES_ONLY"
    }
  }

  expect_failures = [
    var.vpc_access,
  ]
}

# Setting neither is rejected too: an unset vpc_access should stay null
# entirely (the pre-existing "no VPC access at all" shape), not an empty object.
run "neither_network_interfaces_nor_connector_is_rejected" {
  command = plan

  variables {
    vpc_access = {
      egress = "PRIVATE_RANGES_ONLY"
    }
  }

  expect_failures = [
    var.vpc_access,
  ]
}
