# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

# Plan-only tests that pin the rendered Cloud Run billing mode.
#
# cpu_idle = true on a container's resources block keeps the service on
# request-based billing; cpu_idle = false switches it to instance-based
# billing. Consumers usually pass resources objects with cpu_idle unset
# (null), so the coalesce() defaults in main.tf are what decide the rendered
# value. These tests fail when a variable-default or pass-through change
# flips the rendered plan — a change that a consumer PR diff does not show.
#
# Mock providers keep this fully offline: no credentials, no state.

mock_provider "google" {}
mock_provider "google-beta" {}

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

# The main container stays on request-based billing when the consumer does
# not set resources.cpu_idle.
run "main_container_defaults_to_request_based_billing" {
  command = plan

  assert {
    condition     = google_cloud_run_v2_service.this["us-central1"].template[0].containers[0].resources[0].cpu_idle == true
    error_message = "main container rendered cpu_idle != true with cpu_idle unset; the default must stay request-based billing"
  }
}

# An explicit consumer opt-in to instance-based billing is respected.
run "main_container_explicit_cpu_idle_false_is_respected" {
  command = plan

  variables {
    containers = {
      "main" = {
        image = "cgr.dev/chainguard/static:latest"
        ports = [{ container_port = 8080 }]
        resources = {
          cpu_idle = false
        }
      }
    }
  }

  assert {
    condition     = google_cloud_run_v2_service.this["us-central1"].template[0].containers[0].resources[0].cpu_idle == false
    error_message = "main container rendered cpu_idle != false despite an explicit cpu_idle = false"
  }
}

# With otel_resources set but cpu_idle omitted (the shape of
# regional-go-service's default), the otel sidecar must stay on
# request-based billing.
run "otel_sidecar_resources_without_cpu_idle_stay_request_based" {
  command = plan

  variables {
    enable_otel_sidecar = true
    otel_resources = {
      limits = {
        cpu    = "1000m"
        memory = "1Gi"
      }
    }
  }

  assert {
    condition     = google_cloud_run_v2_service.this["us-central1"].template[0].containers[1].resources[0].cpu_idle == true
    error_message = "otel sidecar rendered cpu_idle != true with otel_resources.cpu_idle unset; the default must stay request-based billing"
  }
}

# An explicit otel_resources.cpu_idle = false is respected.
run "otel_sidecar_explicit_cpu_idle_false_is_respected" {
  command = plan

  variables {
    enable_otel_sidecar = true
    otel_resources = {
      limits = {
        cpu    = "1000m"
        memory = "1Gi"
      }
      cpu_idle = false
    }
  }

  assert {
    condition     = google_cloud_run_v2_service.this["us-central1"].template[0].containers[1].resources[0].cpu_idle == false
    error_message = "otel sidecar rendered cpu_idle != false despite an explicit cpu_idle = false"
  }
}

# With otel_resources null, no resources block renders at all and the
# provider default (request-based billing) applies. Set explicitly rather
# than relying on the variable default, so a billing-safe default change
# does not break this run.
run "otel_sidecar_null_resources_render_no_resources_block" {
  command = plan

  variables {
    enable_otel_sidecar = true
    otel_resources      = null
  }

  assert {
    condition     = length(google_cloud_run_v2_service.this["us-central1"].template[0].containers[1].resources) == 0
    error_message = "otel sidecar rendered a resources block with otel_resources = null"
  }
}
