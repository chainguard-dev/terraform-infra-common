# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

# Plan-only guard on this module's billing-relevant defaults.
#
# regional-go-service passes var.otel_resources through to regional-service
# verbatim, and the rendered cpu_idle values are asserted in
# ../regional-service/tests/billing.tftest.hcl. Test assertions cannot
# reference resources inside child modules, so this file pins what this
# layer owns: the otel_resources default must never carry cpu_idle = false,
# which would opt every consumer into instance-based billing.
#
# The regional-service call is replaced with override_module so the plan
# stays fully offline: no credentials, no state. The google providers are
# mocked as well: override_module skips the child module's resources, but
# the providers it requires are still configured, and the real ones would
# try to load application default credentials.

mock_provider "ko" {}
mock_provider "cosign" {}
mock_provider "google" {}
mock_provider "google-beta" {}

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

run "default_otel_resources_do_not_force_instance_based_billing" {
  command = plan

  assert {
    condition     = try(var.otel_resources.cpu_idle, null) != false
    error_message = "the otel_resources default sets cpu_idle = false, opting every consumer into instance-based billing"
  }
}
