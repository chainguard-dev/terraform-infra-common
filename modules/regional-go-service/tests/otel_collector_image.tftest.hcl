# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0
#
# Run in CI by .chainguard/ci/tf-module-tests.yaml.
#
# regional-go-service forwards otel_collector_image to regional-service with
# its own default, so this pins the default this layer owns: it must be
# pinned by digest and keep its tag for digestabot.
# ../regional-service/tests/otel_collector_image.tftest.hcl says why.
#
# The regional-service call is replaced with override_module, as in
# cpu_idle.tftest.hcl, so the plan stays fully offline.

mock_provider "ko" {}
mock_provider "cosign" {}
mock_provider "google-beta" {}

mock_provider "google" {}

override_module {
  target = module.this
  outputs = {
    names     = {}
    locations = {}
    uris      = {}
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
      source = {
        working_dir = "."
        importpath  = "example.com/fixture/cmd/app"
      }
      ports = [{ container_port = 8080 }]
    }
  }
}

run "default_sidecar_image_is_pinned_by_tag_and_digest" {
  command = plan

  assert {
    condition     = can(regex("^[^@]+:[^@/:]+@sha256:[0-9a-f]{64}$", var.otel_collector_image))
    error_message = "The otel_collector_image default ${var.otel_collector_image} must be repository:tag@sha256:<digest>."
  }
}
