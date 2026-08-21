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
mock_provider "google-beta" {}
mock_provider "random" {
  mock_resource "random_string" {
    override_during = plan
    defaults = {
      result = "abc123"
    }
  }
}

variables {
  project_id = "fixture-project"
  name       = "fixture"
  broker     = "projects/fixture-project/topics/fixture-broker"
  private-service = {
    name   = "fixture-service"
    region = "us-central1"
  }
  notification_channels = []
  team                  = "fixture"
}

run "resource_manager_tags_default_to_empty" {
  command = plan
  assert {
    condition     = length(google_tags_tag_binding.dead_letter_topic) == 0
    error_message = "default resource_manager_tags must create no tag bindings"
  }
  assert {
    condition     = length(google_tags_location_tag_binding.dlq_bucket) == 0
    error_message = "the DLQ bucket is disabled by default, so it must bind no tags"
  }
}

# Pub/Sub tag bindings are global: the API rejects a location for these parents,
# so these must stay google_tags_tag_binding and never gain a location.
run "pubsub_tags_bind_globally" {
  command = plan
  variables {
    resource_manager_tags = { "tagKeys/123" = "tagValues/456" }
  }
  assert {
    condition     = google_tags_tag_binding.dead_letter_topic["tagKeys/123"].parent == "//pubsub.googleapis.com/projects/123456789/topics/fixture-dlq-abc123"
    error_message = "Pub/Sub topic tag binding must use the global full resource name with the numeric project number"
  }
  assert {
    condition     = google_tags_tag_binding.subscription["tagKeys/123"].parent == "//pubsub.googleapis.com/projects/123456789/subscriptions/fixture-abc123"
    error_message = "Pub/Sub subscription tag binding must use the global full resource name with the numeric project number"
  }
}

# enable_dlq_bucket is nullable and the bucket treats null as enabled, so the
# binding gate must tolerate null too. A bare `var.enable_dlq_bucket ?` would
# fail here with "The condition value is null".
run "null_enable_dlq_bucket_still_binds" {
  command = plan
  variables {
    enable_dlq_bucket     = null
    resource_manager_tags = { "tagKeys/123" = "tagValues/456" }
  }
  assert {
    condition     = length(google_tags_location_tag_binding.dlq_bucket) == 1
    error_message = "a null enable_dlq_bucket creates the bucket, so it must also bind the tag"
  }
}
