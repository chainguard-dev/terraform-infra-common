# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0
#
# Run in CI by .chainguard/ci/tf-module-tests.yaml.
#
# Cloud Run resolves an image tag whenever it deploys the job, so a tag-only
# sidecar default would make every consumer's deploy depend on the registry
# serving that tag at that moment. The default must name a digest, and keep
# its tag so digestabot can move the digest.

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
    }
  }
}

run "default_sidecar_image_is_pinned_by_tag_and_digest" {
  command = plan

  assert {
    condition     = can(regex("^[^@]+:[^@/:]+@sha256:[0-9a-f]{64}$", google_cloud_run_v2_job.this["us-central1"].template[0].template[0].containers[1].image))
    error_message = "The otel sidecar rendered image ${google_cloud_run_v2_job.this["us-central1"].template[0].template[0].containers[1].image}; the default must be repository:tag@sha256:<digest>."
  }
}
