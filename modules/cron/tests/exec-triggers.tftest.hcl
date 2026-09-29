# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0
#
# Plan-only checks of actual execution triggers. Never apply this fixture:
# null_resource.exec contains the real job-execution provisioner.

mock_provider "google" {}
mock_provider "google-beta" {}
mock_provider "ko" {}
mock_provider "cosign" {}
mock_provider "null" {}

variables {
  project_id            = "fixture-project"
  name                  = "fixture"
  schedule              = "0 * * * *"
  service_account       = "fixture@fixture-project.iam.gserviceaccount.com"
  working_dir           = "."
  importpath            = "example.com/fixture/cmd/app"
  team                  = "fixture"
  notification_channels = []
  exec                  = true
}

override_module {
  override_during = plan
  target          = module.impl
  outputs = {
    job_name = "fixture-cron"
    job_ids  = { "us-east4" = "fixture-job" }
    image_refs = {
      this = "example.invalid/application@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
    }
  }
}

run "default_tracks_images" {
  command = plan

  assert {
    condition     = null_resource.exec[0].triggers == tomap({ "image_refs" = "example.invalid/application@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" })
    error_message = "The planned execution triggers must retain the selected image and policy change signals."
  }
}

run "custom_default_unchanged" {
  command = plan

  variables {
    exec_triggers = { policy = "policy-one" }
  }

  assert {
    condition     = null_resource.exec[0].triggers == tomap({ "policy" = "policy-one" })
    error_message = "The planned execution triggers must retain the selected image and policy change signals."
  }
}

run "legacy_custom_image_key_unchanged" {
  command = plan

  variables {
    exec_triggers = { image_refs = "caller-controlled-signal" }
  }

  assert {
    condition     = null_resource.exec[0].triggers == tomap({ "image_refs" = "caller-controlled-signal" })
    error_message = "The planned execution triggers must retain the selected image and policy change signals."
  }
}

run "combined_tracks_images_and_policy" {
  command = plan

  variables {
    exec_include_image_refs = true
    exec_triggers           = { policy = "policy-one" }
  }

  assert {
    condition     = null_resource.exec[0].triggers == tomap({ "policy" = "policy-one", "image_refs" = "example.invalid/application@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" })
    error_message = "The planned execution triggers must retain the selected image and policy change signals."
  }
}

run "combined_policy_change" {
  command = plan

  variables {
    exec_include_image_refs = true
    exec_triggers           = { policy = "policy-two" }
  }

  assert {
    condition     = null_resource.exec[0].triggers == tomap({ "policy" = "policy-two", "image_refs" = "example.invalid/application@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" })
    error_message = "The planned execution triggers must retain the selected image and policy change signals."
  }
}

run "combined_image_change" {
  command = plan

  variables {
    exec_include_image_refs = true
    exec_triggers           = { policy = "policy-one" }
  }

  override_module {
    override_during = plan
    target          = module.impl
    outputs = {
      job_name = "fixture-cron"
      job_ids  = { "us-east4" = "fixture-job" }
      image_refs = {
        this = "example.invalid/application@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
      }
    }
  }

  assert {
    condition     = null_resource.exec[0].triggers == tomap({ "policy" = "policy-one", "image_refs" = "example.invalid/application@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" })
    error_message = "The planned execution triggers must retain the selected image and policy change signals."
  }
}

run "combined_rejects_reserved_image_key" {
  command = plan

  variables {
    exec_include_image_refs = true
    exec_triggers           = { image_refs = "caller-controlled-signal" }
  }

  expect_failures = [var.exec_include_image_refs]
}

run "disabled_legacy_null_unchanged" {
  command = plan

  variables {
    exec          = false
    exec_triggers = null
  }

  assert {
    condition     = length(null_resource.exec) == 0
    error_message = "Disabled execution must retain its existing empty resource set."
  }
}

run "combined_rejects_null_map" {
  command = plan

  variables {
    exec_include_image_refs = true
    exec_triggers           = null
  }

  expect_failures = [var.exec_include_image_refs]
}
