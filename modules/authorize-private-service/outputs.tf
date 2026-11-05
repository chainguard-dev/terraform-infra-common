output "uri" {
  description = "The URI of the private Cloud Run service."
  value       = data.google_cloud_run_v2_service.this.uri
}

output "invocation_grant" {
  description = "Service invocation grant managed by this module. Excludes inherited and out-of-band IAM grants."
  value = {
    project   = google_cloud_run_v2_service_iam_member.authorize-calls.project
    location  = google_cloud_run_v2_service_iam_member.authorize-calls.location
    name      = google_cloud_run_v2_service_iam_member.authorize-calls.name
    role      = google_cloud_run_v2_service_iam_member.authorize-calls.role
    member    = google_cloud_run_v2_service_iam_member.authorize-calls.member
    condition = google_cloud_run_v2_service_iam_member.authorize-calls.condition
  }
}
