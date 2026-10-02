# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

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
  egress          = "PRIVATE_RANGES_ONLY"
  timeout         = "1200s"
  max_retries     = 2
  task_count      = 3
  parallelism     = 2
  regions = {
    us-central1 = {
      network = "projects/fixture-project/global/networks/fixture"
      subnet  = "projects/fixture-project/regions/us-central1/subnetworks/fixture"
    }
    us-west1 = {}
    us-east1 = {}
  }
  regional-connector = {
    us-west1 = "projects/fixture-project/locations/us-west1/connectors/fixture"
  }
  regional-cronspec = {
    us-central1 = {
      schedule  = "*/5 * * * *"
      time_zone = "UTC"
    }
    us-west1 = {
      schedule  = "*/7 * * * *"
      time_zone = "America/Los_Angeles"
    }
    us-east1 = {
      schedule  = "*/9 * * * *"
      time_zone = "America/New_York"
    }
  }
  containers = {
    main = {
      source = {
        working_dir = "."
        importpath  = "example.com/fixture/cmd/job"
      }
      regional-env = [{
        name = "REGION_MARKER"
        value = {
          us-central1 = "us-central1"
          us-west1    = "us-west1"
          us-east1    = "us-east1"
        }
      }]
    }
  }
}

run "rendered_job_outputs" {
  command = plan
  assert {
    condition = (
      toset(keys(output.execution_policy)) == toset(["us-central1", "us-west1", "us-east1"]) &&
      output.execution_policy["us-central1"].egress == "PRIVATE_RANGES_ONLY" &&
      output.execution_policy["us-west1"].egress == "PRIVATE_RANGES_ONLY" &&
      output.execution_policy["us-east1"].egress == null
    )
    error_message = "Job outputs must report direct and connector VPC egress, and null when neither is configured."
  }
  assert {
    condition = (
      output.execution_policy["us-central1"].schedule == "*/5 * * * *" &&
      output.execution_policy["us-central1"].time_zone == "UTC" &&
      output.execution_policy["us-west1"].schedule == "*/7 * * * *" &&
      output.execution_policy["us-west1"].time_zone == "America/Los_Angeles" &&
      output.execution_policy["us-east1"].schedule == "*/9 * * * *" &&
      output.execution_policy["us-east1"].time_zone == "America/New_York"
    )
    error_message = "Job outputs must preserve each region's schedule and time zone."
  }
  assert {
    condition = alltrue([for policy in output.execution_policy :
      policy.timeout == "1200s" && policy.max_retries == 2 && policy.task_count == 3 && policy.parallelism == 2
    ])
    error_message = "Job outputs must report the rendered execution bounds."
  }
  assert {
    condition = toset(keys(output.containers)) == toset(["us-central1", "us-west1", "us-east1"]) && alltrue([for region, containers in output.containers :
      one(flatten([for container in containers : [for env in container.env : env.value if env.name == "REGION_MARKER"]])) == region
    ])
    error_message = "The exported job containers must contain each region's rendered environment."
  }
}
