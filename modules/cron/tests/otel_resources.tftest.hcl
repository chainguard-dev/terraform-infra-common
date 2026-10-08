# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0
#
# Run in CI by .github/workflows/tf-module-tests.yaml.
#
# otel_resources is passed through to regional-go-cron, so a cron caller can
# size the otel sidecar. regional-go-cron's own tests cover how it resolves
# the value; these runs check that the wrapper forwards it unchanged.

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

run "unset_otel_resources_takes_the_module_default" {
  command = plan

  assert {
    condition     = module.impl.containers[var.region][1].resources[0].limits == tomap({ cpu = "250m", memory = "512Mi" })
    error_message = "An unset otel_resources must leave the sidecar at the regional-go-cron default of 250m CPU and 512Mi memory."
  }
}

run "explicit_otel_resources_are_forwarded" {
  command = plan

  variables {
    otel_resources = { limits = { cpu = "500m", memory = "512Mi" } }
  }

  assert {
    condition     = module.impl.containers[var.region][1].resources[0].limits == tomap({ cpu = "500m", memory = "512Mi" })
    error_message = "Explicit otel_resources limits must reach the sidecar as given."
  }
}

run "null_limits_are_forwarded" {
  command = plan

  variables {
    otel_resources = { limits = null }
  }

  assert {
    condition     = length(module.impl.containers[var.region][1].resources) == 0
    error_message = "otel_resources with null limits must leave the sidecar resources to Cloud Run."
  }
}
