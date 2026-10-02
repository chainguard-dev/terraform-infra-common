// Copyright 2026 Chainguard, Inc.
// SPDX-License-Identifier: Apache-2.0

output "job_name" {
  description = "The name of the Cloud Run Job created in each region."
  value       = var.name

  # This allows callers to refer to `module.[module_name].job_name`, and get
  # ordered after the job's resources are created.
  depends_on = [google_cloud_run_v2_job.this]
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

output "execution_policy" {
  description = "Regional job VPC egress, execution bounds, and scheduler cadence."
  value = {
    for region, job in google_cloud_run_v2_job.this : region => {
      egress      = one(job.template[0].template[0].vpc_access[*].egress)
      timeout     = job.template[0].template[0].timeout
      max_retries = job.template[0].template[0].max_retries
      task_count  = job.template[0].task_count
      parallelism = job.template[0].parallelism
      schedule    = google_cloud_scheduler_job.this[region].schedule
      time_zone   = google_cloud_scheduler_job.this[region].time_zone
    }
  }
}

output "containers" {
  description = "Rendered regional job container configuration, including environment and secret references."
  sensitive   = true
  value       = { for region, job in google_cloud_run_v2_job.this : region => job.template[0].template[0].containers }
}
