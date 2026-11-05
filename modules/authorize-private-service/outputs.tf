output "uri" {
  description = "The URI of the private Cloud Run service."
  value       = data.google_cloud_run_v2_service.this.uri
}

output "deterministic_uri" {
  description = "The private Cloud Run service's deterministic URL, https://<name>-<project number>.<region>.run.app, or null if Cloud Run reports none. A caller that dials it presents an ID token for that audience, which the service can know before it exists."
  # urls holds both the deterministic URL and the hashed *.a.run.app one that
  # uri reports; only the deterministic one ends in the region. try() keeps
  # an unexpected match count from failing modules that never read this.
  value = try(one([
    for url in data.google_cloud_run_v2_service.this.urls : url
    if endswith(url, ".${var.region}.run.app")
  ]), null)
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
