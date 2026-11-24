# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0
#
# Run in CI by .chainguard/ci/tf-module-tests.yaml.
#
# Cloud Run resolves an image tag on every deploy, so a tag-only sidecar
# default would make every consumer's deploy depend on the registry serving
# that tag at that moment. The default must name a digest, and keep its tag
# so digestabot can move the digest.
#
# Mock providers keep this fully offline: no credentials, no state.

mock_provider "google-beta" {}

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
  containers = {
    "main" = {
      image = "cgr.dev/chainguard/static:latest"
      ports = [{ container_port = 8080 }]
    }
  }
}

run "default_sidecar_image_is_pinned_by_tag_and_digest" {
  command = plan

  assert {
    condition     = can(regex("^[^@]+:[^@/:]+@sha256:[0-9a-f]{64}$", google_cloud_run_v2_service.this["us-central1"].template[0].containers[1].image))
    error_message = "The otel sidecar rendered image ${google_cloud_run_v2_service.this["us-central1"].template[0].containers[1].image}; the default must be repository:tag@sha256:<digest>."
  }
}
