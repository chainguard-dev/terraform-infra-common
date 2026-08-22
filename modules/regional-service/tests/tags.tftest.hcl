# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0
# Run in CI by .github/workflows/tf-module-tests.yaml.
#

# Plan-only. Mock providers keep this offline.

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

run "resource_manager_tags_default_to_empty" {
  command = plan

  assert {
    condition     = length(google_tags_location_tag_binding.this) == 0
    error_message = "default resource_manager_tags must create no tag bindings"
  }
}

run "resource_manager_tags_bind_every_region_tag_pair" {
  command = plan

  variables {
    regions = {
      "us-central1" = {
        network = "projects/fixture-project/global/networks/fixture"
        subnet  = "projects/fixture-project/regions/us-central1/subnetworks/fixture"
      }
      "us-east1" = {
        network = "projects/fixture-project/global/networks/fixture"
        subnet  = "projects/fixture-project/regions/us-east1/subnetworks/fixture"
      }
    }
    resource_manager_tags = {
      "tagKeys/123" = "tagValues/456"
      "tagKeys/789" = "tagValues/012"
    }
  }

  assert {
    condition     = length(google_tags_location_tag_binding.this) == 4
    error_message = "two tags on two regions must create four location tag bindings"
  }

  assert {
    condition     = google_tags_location_tag_binding.this["us-east1/tagKeys/789"].location == "us-east1"
    error_message = "regional service tag binding must preserve the region for each cartesian-product pair"
  }

  assert {
    condition     = google_tags_location_tag_binding.this["us-east1/tagKeys/789"].tag_value == "tagValues/012"
    error_message = "regional service tag binding must preserve the tag value for each cartesian-product pair"
  }

  assert {
    condition     = google_tags_location_tag_binding.this["us-east1/tagKeys/789"].parent == "//run.googleapis.com/projects/123456789/locations/us-east1/services/fixture"
    error_message = "Cloud Run tag binding must use the numeric project number"
  }
}

run "resource_manager_tags_reject_malformed_ids" {
  command = plan

  variables {
    resource_manager_tags = {
      "not-a-tag-key" = "not-a-tag-value"
    }
  }

  expect_failures = [var.resource_manager_tags]
}
