# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0
# Run in CI by .chainguard/ci/tf-module-tests.yaml.
#

# Tests that pin which of a service's URLs deterministic_uri reports.
#
# Callers mint ID tokens for the URL they dial, and a service that checks
# audiences knows only its deterministic URL. If the filter matched the
# hashed *.a.run.app URL, or nothing, callers would silently dial the wrong
# URL or get null.
#
# The runs apply against mock providers: the data source depends on the IAM
# member, so a plan defers its read and the output is unknown.

mock_provider "google" {}

variables {
  project_id      = "fixture-project"
  region          = "us-central1"
  name            = "fixture"
  service-account = "fixture@fixture-project.iam.gserviceaccount.com"
}

run "reports_the_regional_url" {
  command = apply

  override_data {
    target = data.google_cloud_run_v2_service.this
    values = {
      uri = "https://fixture-abc123-uc.a.run.app"
      urls = [
        "https://fixture-123456789.us-central1.run.app",
        "https://fixture-abc123-uc.a.run.app",
      ]
    }
  }

  assert {
    condition     = output.deterministic_uri == "https://fixture-123456789.us-central1.run.app"
    error_message = "deterministic_uri is not the service's .us-central1.run.app URL"
  }
}

run "is_null_without_a_regional_url" {
  command = apply

  override_data {
    target = data.google_cloud_run_v2_service.this
    values = {
      uri  = "https://fixture-abc123-uc.a.run.app"
      urls = ["https://fixture-abc123-uc.a.run.app"]
    }
  }

  assert {
    condition     = output.deterministic_uri == null
    error_message = "deterministic_uri matched the hashed *.a.run.app URL"
  }
}

run "ignores_another_regions_url" {
  command = apply

  override_data {
    target = data.google_cloud_run_v2_service.this
    values = {
      uri  = "https://fixture-abc123-uc.a.run.app"
      urls = ["https://fixture-123456789.us-east4.run.app"]
    }
  }

  assert {
    condition     = output.deterministic_uri == null
    error_message = "deterministic_uri reported a URL for a region other than var.region"
  }
}
