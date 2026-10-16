# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0
#
# Run in CI by .github/workflows/tf-module-tests.yaml.

# The mocked lookup returns a different number than the one supplied below, so
# a lookup that leaks back in changes the asserted parent.
mock_provider "google" {
  mock_data "google_project" {
    defaults = {
      number = "123456789"
    }
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
  regions = {
    us-central1 = {}
  }
  regional-cronspec = {
    us-central1 = { schedule = "0 * * * *" }
  }
}

# The wrapper sits behind a module-level depends_on on a resource that is being
# created, so a google_project data source inside the module would be deferred
# and the binding parent would be unknown at plan.
run "depends_on_boundary_keeps_binding_parent_known" {
  command = plan

  module {
    source = "./tests/depends_on_wrapper"
  }

  variables {
    project_number = "987654321"
    resource_manager_tags = {
      "tagKeys/123" = "tagValues/456"
    }
  }

  assert {
    condition     = output.binding_parents["us-central1/tagKeys/123"] == "//run.googleapis.com/projects/987654321/locations/us-central1/jobs/fixture"
    error_message = "tag binding parent must be known at plan across a depends_on boundary and use project_number, not a project lookup"
  }
}

run "project_number_rejects_project_id" {
  command = plan

  variables {
    project_number = "fixture-project"
  }

  expect_failures = [var.project_number]
}

run "project_number_rejects_empty_string" {
  command = plan

  variables {
    project_number = ""
  }

  expect_failures = [var.project_number]
}
