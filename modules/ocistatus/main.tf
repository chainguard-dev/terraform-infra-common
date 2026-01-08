/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

# Artifact Registry repository for OCI statusmanager attestations
resource "google_artifact_registry_repository" "attestations" {
  project       = var.project_id
  location      = var.location
  repository_id = var.name
  description   = "Attestation storage for OCI statusmanager"
  format        = "DOCKER"

  cleanup_policy_dry_run = false

  cleanup_policies {
    id     = "delete-old-untagged"
    action = "DELETE"
    condition {
      tag_state  = "UNTAGGED"
      older_than = var.cleanup_policy_older_than
    }
  }
}

# Grant service account write access to the attestation repository
resource "google_artifact_registry_repository_iam_member" "writer" {
  project    = google_artifact_registry_repository.attestations.project
  location   = google_artifact_registry_repository.attestations.location
  repository = google_artifact_registry_repository.attestations.name
  role       = "roles/artifactregistry.writer"
  member     = var.service_account
}
