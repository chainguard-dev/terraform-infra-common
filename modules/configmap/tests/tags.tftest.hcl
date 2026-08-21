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
  project_id            = "fixture-project"
  name                  = "fixture"
  data                  = "fixture-data"
  service-account       = "fixture@fixture-project.iam.gserviceaccount.com"
  notification-channels = []
  team                  = "fixture"
}

run "resource_manager_tags_default_to_empty" {
  command = plan
  assert {
    condition     = length(google_tags_tag_binding.this) == 0
    error_message = "default resource_manager_tags must create no tag bindings"
  }
}

run "resource_manager_tags_bind_global_configmap_secret" {
  command = plan
  variables {
    resource_manager_tags = { "tagKeys/123" = "tagValues/456" }
  }
  assert {
    condition     = google_tags_tag_binding.this["tagKeys/123"].tag_value == "tagValues/456"
    error_message = "global configmap-secret tag binding must preserve the tag value"
  }
  assert {
    condition     = google_tags_tag_binding.this["tagKeys/123"].parent == "//secretmanager.googleapis.com/projects/123456789/secrets/fixture"
    error_message = "configmap-secret tag binding must use the numeric project number in the documented global full resource name"
  }
}
