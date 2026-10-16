# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0
#
# Run in CI by .github/workflows/tf-module-tests.yaml.
#
# The mocked lookup number differs from the supplied one, so a leaked
# google_project lookup changes the asserted parent.

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

run "project_number_skips_lookup_and_is_used" {
  command = plan

  variables {
    project_number        = "987654321"
    resource_manager_tags = { "tagKeys/123" = "tagValues/456" }
  }

  assert {
    condition     = length(data.google_project.resource_manager_tags) == 0
    error_message = "project_number must skip the google_project data source"
  }
  assert {
    condition = alltrue([
      for k, p in output.tag_binding_parents : startswith(p, "//pubsub.googleapis.com/projects/987654321/")
    ]) && length(output.tag_binding_parents) == 3
    error_message = "every Pub/Sub tag binding parent must use the supplied project_number"
  }
}

run "null_project_number_falls_back_to_lookup" {
  command = plan

  variables {
    resource_manager_tags = { "tagKeys/123" = "tagValues/456" }
  }

  assert {
    condition     = length(data.google_project.resource_manager_tags) == 1
    error_message = "a null project_number must read the google_project data source"
  }
  assert {
    condition = alltrue([
      for k, p in output.tag_binding_parents : startswith(p, "//pubsub.googleapis.com/projects/123456789/")
    ]) && length(output.tag_binding_parents) == 3
    error_message = "parents must use the looked-up number when project_number is null"
  }
}

run "empty_tags_bind_nothing_and_skip_lookup" {
  command = plan

  assert {
    condition     = length(output.tag_binding_parents) == 0
    error_message = "empty tags must create no tag bindings"
  }
  assert {
    condition     = length(data.google_project.resource_manager_tags) == 0
    error_message = "empty tags must not read the google_project data source"
  }
}

run "non_numeric_project_number_is_rejected" {
  command = plan

  variables {
    project_number = "fixture-project"
  }

  expect_failures = [var.project_number]
}

# The module is called behind a module-level depends_on on a resource that is
# being created, which defers every data source read to apply.
run "depends_on_wrapper_parents_are_known_at_plan" {
  command = plan

  module {
    source = "./tests/depends_on_wrapper"
  }

  variables {
    project_number        = "987654321"
    resource_manager_tags = { "tagKeys/123" = "tagValues/456" }
  }

  assert {
    condition = alltrue([
      for k, p in output.tag_binding_parents : startswith(p, "//pubsub.googleapis.com/projects/987654321/")
    ]) && length(output.tag_binding_parents) == 3
    error_message = "tag binding parents must be known at plan and use project_number behind depends_on"
  }
}
