# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

mock_provider "google" {
  mock_data "google_project" {
    defaults = {
      number = "123456789"
    }
  }
  mock_resource "google_artifact_registry_repository" {
    defaults = {
      name         = "projects/fixture-project/locations/us-central1/repositories/fixture"
      registry_uri = "us-central1-docker.pkg.dev/fixture-project/fixture"
    }
  }
}

variables {
  project_id      = "fixture-project"
  name            = "fixture"
  location        = "us-central1"
  service_account = "serviceAccount:fixture@fixture-project.iam.gserviceaccount.com"
}

run "resource_manager_tags_default_to_empty" {
  command = plan
  assert {
    condition     = length(google_tags_location_tag_binding.attestations) == 0
    error_message = "default resource_manager_tags must create no tag bindings"
  }
}

run "resource_manager_tags_bind_artifact_registry" {
  command = plan
  variables {
    resource_manager_tags = { "tagKeys/123" = "tagValues/456" }
  }
  assert {
    condition     = google_tags_location_tag_binding.attestations["tagKeys/123"].location == "us-central1"
    error_message = "Artifact Registry tag binding must use the repository location"
  }
  assert {
    condition     = google_tags_location_tag_binding.attestations["tagKeys/123"].parent == "//artifactregistry.googleapis.com/projects/123456789/locations/us-central1/repositories/fixture"
    error_message = "Artifact Registry tag binding must use the numeric project number in the verified full resource name"
  }
}

run "malformed_tags_are_rejected" {
  command = plan
  variables {
    resource_manager_tags = { "tagKeys/nope" = "tagValues/nope" }
  }
  expect_failures = [var.resource_manager_tags]
}

run "attestation_repository_applies_with_writer_and_tags" {
  command = apply
  variables {
    resource_manager_tags     = { "tagKeys/123" = "tagValues/456" }
    cleanup_policy_older_than = "1d"
  }
  assert {
    condition = (
      output.repository_id == "fixture" &&
      output.registry_uri == google_artifact_registry_repository.attestations.registry_uri &&
      output.attestations_path == "${google_artifact_registry_repository.attestations.registry_uri}/attestations"
    )
    error_message = "The outputs must expose the created repository and its attestations path."
  }
  assert {
    condition = (
      google_artifact_registry_repository_iam_member.writer.repository == google_artifact_registry_repository.attestations.name &&
      google_artifact_registry_repository_iam_member.writer.location == "us-central1" &&
      google_artifact_registry_repository_iam_member.writer.project == "fixture-project" &&
      google_artifact_registry_repository_iam_member.writer.role == "roles/artifactregistry.repoAdmin" &&
      google_artifact_registry_repository_iam_member.writer.member == var.service_account
    )
    error_message = "The writer must get repoAdmin on the created repository."
  }
  assert {
    condition     = [for p in google_artifact_registry_repository.attestations.cleanup_policies : p.condition[0].older_than if p.id == "delete-old-untagged"] == ["1d"]
    error_message = "cleanup_policy_older_than must reach the untagged cleanup policy."
  }
  assert {
    condition     = google_tags_location_tag_binding.attestations["tagKeys/123"].tag_value == "tagValues/456" && endswith(google_tags_location_tag_binding.attestations["tagKeys/123"].parent, "/repositories/${google_artifact_registry_repository.attestations.repository_id}")
    error_message = "Each tag must bind its value to the created repository."
  }
}

run "signed_older_than_is_rejected" {
  command = plan
  variables {
    cleanup_policy_older_than = "-12h"
  }
  expect_failures = [var.cleanup_policy_older_than]
}

run "empty_older_than_is_rejected" {
  command = plan
  variables {
    cleanup_policy_older_than = ""
  }
  expect_failures = [var.cleanup_policy_older_than]
}

run "null_older_than_takes_the_default" {
  command = plan
  variables {
    cleanup_policy_older_than = null
  }
  assert {
    condition     = [for p in google_artifact_registry_repository.attestations.cleanup_policies : p.condition[0].older_than if p.id == "delete-old-untagged"] == ["86400s"]
    error_message = "A null cleanup_policy_older_than must fall back to the 86400s default."
  }
}

run "zero_older_than_is_rejected" {
  command = plan
  variables {
    cleanup_policy_older_than = "0s"
  }
  expect_failures = [var.cleanup_policy_older_than]
}

run "overflowing_older_than_is_rejected" {
  command = plan
  variables {
    cleanup_policy_older_than = "106751991167301d"
  }
  expect_failures = [var.cleanup_policy_older_than]
}

run "fractional_older_than_is_rejected" {
  command = plan
  variables {
    cleanup_policy_older_than = "1.5d"
  }
  expect_failures = [var.cleanup_policy_older_than]
}

run "older_than_with_an_unknown_unit_is_rejected" {
  command = plan
  variables {
    cleanup_policy_older_than = "2w"
  }
  expect_failures = [var.cleanup_policy_older_than]
}

run "older_than_without_a_unit_is_rejected" {
  command = plan
  variables {
    cleanup_policy_older_than = "86400"
  }
  expect_failures = [var.cleanup_policy_older_than]
}

run "namespaced_tag_value_is_rejected" {
  command = plan
  variables {
    resource_manager_tags = { "tagKeys/123" = "fixture-org/env/prod" }
  }
  expect_failures = [var.resource_manager_tags]
}

run "fifty_tags_plan_cleanly" {
  command = plan
  variables {
    resource_manager_tags = { for i in range(50) : "tagKeys/${i}" => "tagValues/${i}" }
  }
  assert {
    condition     = length(google_tags_location_tag_binding.attestations) == 50
    error_message = "Fifty tags must bind."
  }
}

run "fifty_one_tags_are_rejected" {
  command = plan
  variables {
    resource_manager_tags = { for i in range(51) : "tagKeys/${i}" => "tagValues/${i}" }
  }
  expect_failures = [var.resource_manager_tags]
}

run "malformed_tag_key_is_rejected" {
  command = plan
  variables {
    resource_manager_tags = { "fixture-org/env" = "tagValues/456" }
  }
  expect_failures = [var.resource_manager_tags]
}
