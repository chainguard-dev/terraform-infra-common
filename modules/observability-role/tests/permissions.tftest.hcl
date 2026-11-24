# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0
#
# The role stands in for three predefined roles, so its permissions must stay
# exactly their union, and its id must be known at plan for callers' count.

mock_provider "google" {
  mock_resource "google_project_iam_custom_role" {
    defaults = {
      name = "projects/fixture-project/roles/serviceObservability"
    }
  }
}

variables {
  project_id = "fixture-project"
}

run "default_role_unions_observability_permissions" {
  command = plan

  assert {
    condition = (
      google_project_iam_custom_role.this.project == "fixture-project" &&
      google_project_iam_custom_role.this.role_id == "serviceObservability" &&
      google_project_iam_custom_role.this.title == "Service Observability"
    )
    error_message = "the role must default to the serviceObservability id and its title."
  }
  assert {
    condition = toset(google_project_iam_custom_role.this.permissions) == toset([
      "monitoring.metricDescriptors.create",
      "monitoring.metricDescriptors.get",
      "monitoring.metricDescriptors.list",
      "monitoring.monitoredResourceDescriptors.get",
      "monitoring.monitoredResourceDescriptors.list",
      "monitoring.timeSeries.create",
      "cloudtrace.traces.patch",
      "telemetry.traces.write",
      "cloudprofiler.profiles.create",
      "cloudprofiler.profiles.update",
    ])
    error_message = "permissions must be exactly those of monitoring.metricWriter, cloudtrace.agent, and cloudprofiler.agent."
  }
  assert {
    condition     = output.id == "projects/fixture-project/roles/serviceObservability"
    error_message = "id must be the fully qualified role id, known at plan."
  }
}

run "custom_role_id_reaches_output" {
  command = plan

  variables {
    role_id = "fixtureObservability"
  }

  assert {
    condition     = output.id == "projects/fixture-project/roles/fixtureObservability"
    error_message = "id must follow role_id."
  }
}

run "title_at_100_characters_plans" {
  command = plan

  variables {
    title = replace(format("%100s", ""), " ", "a")
  }

  assert {
    condition     = length(google_project_iam_custom_role.this.title) == 100
    error_message = "a 100 character title must plan."
  }
}

run "apply_id_matches_created_role_name" {
  command = apply

  assert {
    condition     = output.id == google_project_iam_custom_role.this.name
    error_message = "id must equal the created role's resource name."
  }
}

run "title_over_100_characters_is_rejected" {
  command = plan

  variables {
    title = replace(format("%101s", ""), " ", "a")
  }

  expect_failures = [var.title]
}

run "title_at_100_bytes_of_two_byte_characters_plans" {
  command = plan

  variables {
    title = replace(format("%50s", ""), " ", "é")
  }

  assert {
    condition     = google_project_iam_custom_role.this.title == replace(format("%50s", ""), " ", "é")
    error_message = "a title of 50 two-byte characters (100 bytes) must plan."
  }
}

run "title_at_100_bytes_of_four_byte_characters_plans" {
  command = plan

  variables {
    title = replace(format("%25s", ""), " ", "\U0001F600")
  }

  assert {
    condition     = length(google_project_iam_custom_role.this.title) == 25
    error_message = "a title of 25 four-byte characters (100 bytes) must plan."
  }
}

run "title_over_100_bytes_in_fewer_characters_is_rejected" {
  command = plan

  variables {
    title = replace(format("%51s", ""), " ", "é")
  }

  expect_failures = [var.title]
}

run "title_one_byte_over_with_a_trailing_two_byte_character_is_rejected" {
  command = plan

  variables {
    title = "${replace(format("%99s", ""), " ", "a")}é"
  }

  expect_failures = [var.title]
}

run "title_over_100_bytes_of_four_byte_characters_is_rejected" {
  command = plan

  variables {
    title = replace(format("%26s", ""), " ", "\U0001F600")
  }

  expect_failures = [var.title]
}
