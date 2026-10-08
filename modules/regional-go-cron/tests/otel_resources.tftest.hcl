# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0
#
# Run in CI by .github/workflows/tf-module-tests.yaml.
#
# Without explicit limits Cloud Run gives the otel sidecar 1 vCPU. The default
# sidecar limit is 250m, raised when the application containers total under
# 750m because gen2 jobs reject a task under 1 vCPU.

mock_provider "google" {
  mock_data "google_project" {
    defaults = { number = "123456789" }
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
  regions         = { us-central1 = {} }
  regional-cronspec = {
    us-central1 = { schedule = "0 0 9 1 *", paused = true }
  }
  containers = {
    this = {
      source = {
        working_dir = "."
        importpath  = "example.com/fixture"
      }
      resources = { limits = { cpu = "2", memory = "8Gi" } }
    }
  }
}

run "default_sidecar_limits" {
  command = plan
  assert {
    condition     = google_cloud_run_v2_job.this["us-central1"].template[0].template[0].containers[1].resources[0].limits == tomap({ cpu = "250m", memory = "512Mi" })
    error_message = "The otel sidecar must default to 250m CPU and 512Mi memory."
  }
}

run "default_sidecar_with_unset_application_cpu" {
  command = plan
  variables {
    containers = {
      this = {
        source = {
          working_dir = "."
          importpath  = "example.com/fixture"
        }
      }
    }
  }
  assert {
    condition     = google_cloud_run_v2_job.this["us-central1"].template[0].template[0].containers[1].resources[0].limits["cpu"] == "250m"
    error_message = "An application container without a cpu limit runs at 1 vCPU, so the sidecar needs no more than 250m."
  }
}

run "default_sidecar_fills_task_to_one_vcpu" {
  command = plan
  variables {
    containers = {
      this = {
        source = {
          working_dir = "."
          importpath  = "example.com/fixture"
        }
        resources = { limits = { cpu = "500m", memory = "1Gi" } }
      }
    }
  }
  assert {
    condition     = google_cloud_run_v2_job.this["us-central1"].template[0].template[0].containers[1].resources[0].limits["cpu"] == "500m"
    error_message = "The sidecar must bring a 500m application container up to the 1 vCPU task minimum."
  }
}

run "explicit_sidecar_limits" {
  command = plan
  variables {
    otel_resources = { limits = { cpu = "500m", memory = "1Gi" } }
  }
  assert {
    condition     = google_cloud_run_v2_job.this["us-central1"].template[0].template[0].containers[1].resources[0].limits == tomap({ cpu = "500m", memory = "1Gi" })
    error_message = "Explicit otel_resources limits must be applied to the sidecar as given."
  }
}

run "null_sidecar_limits_omit_resources" {
  command = plan
  variables {
    otel_resources = { limits = null }
  }
  assert {
    condition     = length(google_cloud_run_v2_job.this["us-central1"].template[0].template[0].containers[1].resources) == 0
    error_message = "otel_resources with null limits must leave the sidecar resources to Cloud Run."
  }
}

run "sidecar_counts_toward_task_cpu_cap" {
  command = plan
  variables {
    containers = {
      this = {
        source = {
          working_dir = "."
          importpath  = "example.com/fixture"
        }
        resources = { limits = { cpu = "8", memory = "32Gi" } }
      }
    }
  }
  expect_failures = [google_cloud_run_v2_job.this]
}

run "explicit_sidecar_cpu_counts_toward_task_cpu_cap" {
  command = plan
  variables {
    containers = {
      this = {
        source = {
          working_dir = "."
          importpath  = "example.com/fixture"
        }
        resources = { limits = { cpu = "7500m", memory = "28Gi" } }
      }
    }
    otel_resources = { limits = { cpu = "1", memory = "512Mi" } }
  }
  expect_failures = [google_cloud_run_v2_job.this]
}

run "explicit_sidecar_cpu_below_task_minimum" {
  command = plan
  variables {
    containers = {
      this = {
        source = {
          working_dir = "."
          importpath  = "example.com/fixture"
        }
        resources = { limits = { cpu = "500m", memory = "1Gi" } }
      }
    }
    otel_resources = { limits = { cpu = "250m", memory = "512Mi" } }
  }
  expect_failures = [google_cloud_run_v2_job.this]
}
