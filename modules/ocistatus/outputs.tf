/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

output "repository_id" {
  description = "The ID of the Artifact Registry repository."
  value       = google_artifact_registry_repository.attestations.repository_id
}

output "registry_uri" {
  description = "The registry URI of the Artifact Registry repository."
  value       = google_artifact_registry_repository.attestations.registry_uri
}

output "attestations_path" {
  description = "The full path for storing attestations (for use with STATUSMANAGER_REPOSITORY)."
  value       = "${google_artifact_registry_repository.attestations.registry_uri}/attestations"
}
