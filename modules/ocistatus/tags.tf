// Copyright 2026 Chainguard, Inc.
// SPDX-License-Identifier: Apache-2.0

// The binding parent needs the numeric project number. Read only when tags are set.
data "google_project" "resource_manager_tags" {
  count = length(var.resource_manager_tags) > 0 ? 1 : 0

  project_id = var.project_id
}

// Artifact Registry tag bindings are location scoped to the repository
// location. The parent must be the repository's permanent resource name, so it
// carries the numeric project number, not the project ID.
resource "google_tags_location_tag_binding" "attestations" {
  for_each = var.resource_manager_tags

  parent    = "//artifactregistry.googleapis.com/projects/${data.google_project.resource_manager_tags[0].number}/locations/${var.location}/repositories/${google_artifact_registry_repository.attestations.repository_id}"
  tag_value = each.value
  location  = var.location
}
