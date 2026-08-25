# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

# Offline tests for Cloud Run's 1 vCPU instance-total floor.

mock_provider "google" {}

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
}

run "gen2_app_500m_plus_default_sidecar_is_rejected" {
  command = plan

  variables {
    enable_otel_sidecar = true
    scaling             = { max_instance_request_concurrency = 80 }
    containers = {
      "main" = {
        image     = "cgr.dev/chainguard/static:latest"
        ports     = [{ container_port = 8080 }]
        resources = { limits = { cpu = "500m", memory = "512Mi" } }
      }
    }
  }

  expect_failures = [google_cloud_run_v2_service.this]
}

run "app_750m_plus_default_sidecar_reaches_the_floor" {
  command = plan

  variables {
    enable_otel_sidecar = true
    scaling             = { max_instance_request_concurrency = 80 }
    containers = {
      "main" = {
        image     = "cgr.dev/chainguard/static:latest"
        ports     = [{ container_port = 8080 }]
        resources = { limits = { cpu = "750m", memory = "512Mi" } }
      }
    }
  }

  assert {
    condition     = local.total_effective_millicpu == 1000
    error_message = "750m app plus the 250m sidecar default should reach the 1000m floor, got ${local.total_effective_millicpu}m"
  }
}

run "gen2_single_concurrency_is_rejected" {
  command = plan

  variables {
    enable_otel_sidecar = true
    scaling             = { max_instance_request_concurrency = 1 }
    containers = {
      "main" = {
        image     = "cgr.dev/chainguard/static:latest"
        ports     = [{ container_port = 8080 }]
        resources = { limits = { cpu = "500m", memory = "512Mi" } }
      }
    }
  }

  expect_failures = [google_cloud_run_v2_service.this]
}

run "gen1_single_concurrency_is_exempt" {
  command = plan

  variables {
    enable_otel_sidecar   = true
    execution_environment = "EXECUTION_ENVIRONMENT_GEN1"
    scaling               = { max_instance_request_concurrency = 1 }
    containers = {
      "main" = {
        image     = "cgr.dev/chainguard/static:latest"
        ports     = [{ container_port = 8080 }]
        resources = { limits = { cpu = "500m", memory = "512Mi" } }
      }
    }
  }

  assert {
    condition     = local.total_effective_millicpu == 750
    error_message = "expected 750m total at Gen1 concurrency 1, got ${local.total_effective_millicpu}m"
  }
}

run "unset_concurrency_uses_cloud_run_default" {
  command = plan

  variables {
    enable_otel_sidecar = true
    containers = {
      "main" = {
        image     = "cgr.dev/chainguard/static:latest"
        ports     = [{ container_port = 8080 }]
        resources = { limits = { cpu = "500m", memory = "512Mi" } }
      }
    }
  }

  expect_failures = [google_cloud_run_v2_service.this]
}

run "omitted_app_cpu_uses_cloud_run_default" {
  command = plan

  variables {
    enable_otel_sidecar = true
    containers = {
      "main" = {
        image = "cgr.dev/chainguard/static:latest"
        ports = [{ container_port = 8080 }]
      }
    }
  }

  assert {
    condition     = local.total_effective_millicpu == 1250
    error_message = "omitted app CPU plus the 250m sidecar should total 1250m, got ${local.total_effective_millicpu}m"
  }
}

run "omitted_otel_cpu_uses_cloud_run_default" {
  command = plan

  variables {
    enable_otel_sidecar = true
    otel_resources      = {}
    containers = {
      "main" = {
        image     = "cgr.dev/chainguard/static:latest"
        ports     = [{ container_port = 8080 }]
        resources = { limits = { cpu = "500m", memory = "512Mi" } }
      }
    }
  }

  assert {
    condition     = local.total_effective_millicpu == 1500
    error_message = "500m app plus an omitted sidecar CPU should total 1500m, got ${local.total_effective_millicpu}m"
  }
}
