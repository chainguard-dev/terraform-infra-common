output "uri" {
  description = "The URI of the private Cloud Run service."
  value       = data.google_cloud_run_v2_service.this.uri
}
