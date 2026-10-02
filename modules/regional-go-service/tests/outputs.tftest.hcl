# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

# Assert on module outputs so callers can rely on the rendered configuration.

mock_provider "google-beta" {}
mock_provider "ko" {}
mock_provider "cosign" {}

mock_provider "google" {}

variables {
  project_id = "fixture-project"
  name       = "fixture"
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
  ingress = "INGRESS_TRAFFIC_ALL"
  regional-egress = {
    us-central1 = "PRIVATE_RANGES_ONLY"
  }
  service_account       = "fixture@fixture-project.iam.gserviceaccount.com"
  notification_channels = []
  team                  = "fixture"
  containers = {
    "main" = {
      source = {
        working_dir = "."
        importpath  = "example.com/fixture/cmd/app"
      }
      ports = [{ container_port = 8080 }]
      regional-env = [{
        name = "REGION_MARKER"
        value = {
          us-central1 = "us-central1"
          us-west1    = "us-west1"
        }
      }]
    }
  }
}

run "forward_rendered_service_outputs" {
  command = plan
  assert {
    condition = (
      toset(keys(output.access_policy)) == toset(["us-central1", "us-west1"]) &&
      alltrue([for policy in output.access_policy : policy.public_invoker && policy.ingress == "INGRESS_TRAFFIC_ALL"]) &&
      output.access_policy["us-central1"].egress == "PRIVATE_RANGES_ONLY" &&
      output.access_policy["us-west1"].egress == "ALL_TRAFFIC"
    )
    error_message = "Public ingress without an authentication requirement must report its public IAM grant, with egress overrides confined to their region."
  }
  assert {
    condition = toset(keys(output.containers)) == toset(["us-central1", "us-west1"]) && alltrue([for region, containers in output.containers :
      one(flatten([for container in containers : [for env in container.env : env.value if env.name == "REGION_MARKER"]])) == region
    ])
    error_message = "The exported containers must contain each region's rendered environment."
  }
}
