// Copyright 2026 Chainguard, Inc.
// SPDX-License-Identifier: Apache-2.0

output "job_name" {
  description = "The name of the Cloud Run Job created in each region."
  value       = var.name
}

output "job_etag" {
  description = "The etag of the Cloud Run Job in each region, changes whenever the job definition changes."
  value       = { for k, v in google_cloud_run_v2_job.this : k => v.etag }
}

output "job_ids" {
  description = "The ID of the Cloud Run Job in each region."
  value       = { for k, v in google_cloud_run_v2_job.this : k => v.id }
}

output "image_refs" {
  description = "The signed image reference for each container, keyed by container name. Computed by ko/cosign before the Cloud Run Job is updated, so stable during apply."
  value       = { for k, v in cosign_sign.this : k => v.signed_ref }
}
