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

run "default_tracks_images_and_job_config" {
  command = plan

  assert {
    condition = null_resource.exec[0].triggers == tomap({
      "image_refs" = "example.invalid/application@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
      "job_config" = sha256(jsonencode({ args = [], command = [], env = {}, secret_env = {}, volume_mounts = [], volumes = [] }))
    })
    error_message = "The planned execution triggers must retain the selected image and job settings change signals."
  }
}

run "default_job_config_tracks_env" {
  command = plan

  variables {
    env = { GRANTEES = "a,b" }
  }

  assert {
    condition     = null_resource.exec[0].triggers["job_config"] == sha256(jsonencode({ args = [], command = [], env = { GRANTEES = "a,b" }, secret_env = {}, volume_mounts = [], volumes = [] }))
    error_message = "An env change must change the default execution triggers, even when the image does not."
  }
}

run "default_job_config_tracks_volumes" {
  command = plan

  variables {
    volumes = [{
      name   = "config"
      secret = { secret = "fixture-config", items = [{ version = "2", path = "config.yaml" }] }
    }]
    volume_mounts = [{ name = "config", mount_path = "/var/run/config" }]
  }

  assert {
    condition     = null_resource.exec[0].triggers["job_config"] != sha256(jsonencode({ args = [], command = [], env = {}, secret_env = {}, volume_mounts = [], volumes = [] }))
    error_message = "A config file mounted from a secret volume must change the default execution triggers, even when the image does not."
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

run "additional_preserves_defaults" {
  command = plan
  variables { exec_additional_triggers = { checkpoint = "one" } }
  assert {
    condition = null_resource.exec[0].triggers == tomap({
      image_refs = "example.invalid/application@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
      job_config = sha256(jsonencode({ args = [], command = [], env = {}, secret_env = {}, volume_mounts = [], volumes = [] }))
      checkpoint = "one"
    })
    error_message = "An additive signal must retain both default image and job-setting signals."
  }
}

run "additional_preserves_custom" {
  command = plan
  variables {
    exec_triggers            = { policy = "policy-one" }
    exec_additional_triggers = { checkpoint = "one" }
  }
  assert {
    condition     = null_resource.exec[0].triggers == tomap({ policy = "policy-one", checkpoint = "one" })
    error_message = "An additive signal must preserve a custom execution signal."
  }
}

run "additional_preserves_custom_and_images" {
  command = plan
  variables {
    exec_triggers            = { policy = "policy-one" }
    exec_include_image_refs  = true
    exec_additional_triggers = { checkpoint = "one" }
  }
  assert {
    condition     = null_resource.exec[0].triggers == tomap({ policy = "policy-one", image_refs = "example.invalid/application@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", checkpoint = "one" })
    error_message = "An additive signal must preserve the selected custom and image signals."
  }
}

run "additional_signal_changes" {
  command = plan
  variables { exec_additional_triggers = { checkpoint = "two" } }
  assert {
    condition     = null_resource.exec[0].triggers["checkpoint"] == "two" && length(null_resource.exec[0].triggers) == 3
    error_message = "Changing the additive signal must change the execution triggers."
  }
}

run "additional_rejects_default_image_collision" {
  command = plan
  variables { exec_additional_triggers = { image_refs = "replacement" } }
  expect_failures = [null_resource.exec]
}

run "additional_rejects_default_job_collision" {
  command = plan
  variables { exec_additional_triggers = { job_config = "replacement" } }
  expect_failures = [null_resource.exec]
}

run "additional_rejects_custom_collision" {
  command = plan
  variables {
    exec_triggers            = { policy = "policy-one" }
    exec_additional_triggers = { policy = "replacement" }
  }
  expect_failures = [null_resource.exec]
}

run "additional_rejects_included_image_collision" {
  command = plan
  variables {
    exec_triggers            = { policy = "policy-one" }
    exec_include_image_refs  = true
    exec_additional_triggers = { image_refs = "replacement" }
  }
  expect_failures = [null_resource.exec]
}
