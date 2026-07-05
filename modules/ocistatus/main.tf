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

# Grant the service account access to manage attestations in the repository.
# The statusmanager resolves write conflicts by deleting superseded attestation
# referrers before writing the replacement (referrer manifests are
# content-addressed, so replacement is delete+write rather than an in-place
# overwrite). That manifest DELETE requires
# artifactregistry.repositories.deleteArtifacts, which is in repoAdmin but not
# writer.
resource "google_artifact_registry_repository_iam_member" "writer" {
  project    = google_artifact_registry_repository.attestations.project
  location   = google_artifact_registry_repository.attestations.location
  repository = google_artifact_registry_repository.attestations.name
  role       = "roles/artifactregistry.repoAdmin"
  member     = var.service_account
}
