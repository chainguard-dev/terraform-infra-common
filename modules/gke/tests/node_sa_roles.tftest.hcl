# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0
#
# Run in CI by the tf-module-tests check in .chainguard/ci/tf-module-tests.yaml.

mock_provider "google" {
  mock_resource "google_container_cluster" {
    override_during = plan
    defaults = {
      id       = "projects/fixture-project/locations/us-central1/clusters/fixture"
      location = "us-central1"
    }
  }
}

mock_provider "google-beta" {}

variables {
  name       = "fixture"
  project    = "fixture-project"
  network    = "fixture-network"
  region     = "us-central1"
  team       = "fixture"
  subnetwork = "fixture-subnetwork"
  pools = {
    plain = {}
  }
}

run "storage_object_viewer_granted_by_default" {
  command = plan

  assert {
    condition     = google_project_iam_member.cluster["gcr"].role == "roles/storage.objectViewer"
    error_message = "a caller that sets nothing must keep the project-level storage.objectViewer grant under the gcr key"
  }

  assert {
    condition     = google_project_iam_member.cluster["acr"].role == "roles/artifactregistry.reader"
    error_message = "the node service account must keep roles/artifactregistry.reader"
  }
}

run "storage_object_viewer_can_be_dropped" {
  command = plan

  variables {
    grant_project_storage_object_viewer = false
  }

  assert {
    condition     = !contains(keys(google_project_iam_member.cluster), "gcr")
    error_message = "grant_project_storage_object_viewer = false must drop the gcr grant"
  }

  assert {
    condition     = google_project_iam_member.cluster["acr"].role == "roles/artifactregistry.reader"
    error_message = "dropping the gcr grant must keep roles/artifactregistry.reader"
  }

  assert {
    condition     = length(google_project_iam_member.cluster) == 4
    error_message = "dropping the gcr grant must leave the other four default roles in place"
  }
}
