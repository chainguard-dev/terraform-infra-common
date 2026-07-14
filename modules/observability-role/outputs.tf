/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

output "id" {
  description = "Fully-qualified role id (projects/{project}/roles/{role_id}), for use as the observability_role input of the service modules."

  // Assembled from configuration attributes rather than the resource's
  // computed id so the value is known at plan time: consumers feed it into
  // count expressions, which cannot depend on unknown values. Referencing
  // the resource attributes still gives Terraform the dependency edge that
  // orders role creation before any grant of the role.
  value = "projects/${google_project_iam_custom_role.this.project}/roles/${google_project_iam_custom_role.this.role_id}"
}
