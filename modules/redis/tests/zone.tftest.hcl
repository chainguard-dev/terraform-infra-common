# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0
#
# Run in CI by .github/workflows/tf-module-tests.yaml.

mock_provider "google" {
  mock_data "google_project" {
    defaults = {
      number = "123456789"
    }
  }
}

variables {
  project_id               = "fixture-project"
  name                     = "fixture"
  team                     = "fixture"
  region                   = "us-central1"
  secret_accessor_sa_email = "fixture@fixture-project.iam.gserviceaccount.com"
  notification_channels    = []
  secret_version_adder     = "serviceAccount:fixture@fixture-project.iam.gserviceaccount.com"
}

run "zone_pins_the_primary_node" {
  command = plan
  variables {
    zone = "us-central1-f"
  }
  assert {
    condition     = google_redis_instance.default.location_id == "us-central1-f"
    error_message = "a zone passed to the module must be sent as location_id"
  }
}

# Memorystore rejects an empty location_id, so an unset zone must reach the API
# as an absent field. On apply the mock provider fills an absent computed
# attribute with a generated value, so location_id is empty only if the module
# turned the unset zone into an empty string.
run "unset_zone_leaves_the_choice_to_memorystore" {
  command = apply
  assert {
    condition     = google_redis_instance.default.location_id != ""
    error_message = "an unset zone must be sent as an unset location_id"
  }
}
