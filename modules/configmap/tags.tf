// Copyright 2026 Chainguard, Inc.
// SPDX-License-Identifier: Apache-2.0

resource "google_tags_tag_binding" "this" {
  for_each = var.resource_manager_tags

  parent    = "//secretmanager.googleapis.com/projects/${data.google_project.project.number}/secrets/${google_secret_manager_secret.this.secret_id}"
  tag_value = each.value
}
