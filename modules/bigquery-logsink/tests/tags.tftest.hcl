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
  project_id = "fixture-project"
  name       = "fixture"
  sinks      = {}
}

run "resource_manager_tags_default_to_empty" {
  command = plan
  assert {
    condition     = length(google_tags_location_tag_binding.dataset) == 0
    error_message = "default resource_manager_tags must create no tag bindings"
  }
}

# nullable = false means an explicit null resolves to the empty default instead
# of blowing up the for_each. Callers forwarding an optional map rely on this.
run "explicit_null_is_inert" {
  command = plan
  variables {
    resource_manager_tags = null
  }
  assert {
    condition     = length(google_tags_location_tag_binding.dataset) == 0
    error_message = "an explicit null must resolve to the empty default and bind nothing"
  }
}

# The module defaults location to US. Tagging a multi-region dataset hits
# hashicorp/terraform-provider-google#18254, so it must fail at plan. The guard
# lives on the variable, not on each binding, because every binding tested this
# same input.
run "multi_region_location_is_rejected_when_tagging" {
  command = plan
  variables {
    resource_manager_tags = { "tagKeys/123" = "tagValues/456" }
  }
  expect_failures = [var.resource_manager_tags]
}

run "single_region_location_binds" {
  command = plan
  variables {
    location              = "us-central1"
    resource_manager_tags = { "tagKeys/123" = "tagValues/456" }
  }
  assert {
    condition     = google_tags_location_tag_binding.dataset["tagKeys/123"].parent == "//bigquery.googleapis.com/projects/123456789/datasets/log_sinks_fixture"
    error_message = "BigQuery dataset tag binding must use the dataset full resource name with the numeric project number"
  }
  assert {
    condition     = google_tags_location_tag_binding.dataset["tagKeys/123"].location == "us-central1"
    error_message = "BigQuery tag binding must use the dataset location"
  }
}
