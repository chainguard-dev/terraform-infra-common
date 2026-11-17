output "secret_id" {
  description = "The ID of the secret."
  value       = google_secret_manager_secret.this.id
}

output "authorized_secret_id" {
  description = "The ID of the secret after its accessor binding is applied and, with bootstrap_placeholder_version, after the placeholder version exists, so a service that reads `latest` can deploy against it."
  value       = google_secret_manager_secret_iam_binding.authorize-service-access.secret_id
  depends_on  = [google_secret_manager_secret_version.bootstrap]
}

output "annotations" {
  description = "The annotations set on the secret."
  value       = google_secret_manager_secret.this.annotations
}

output "accessor_members" {
  description = "The members granted roles/secretmanager.secretAccessor on the secret."
  value       = google_secret_manager_secret_iam_binding.authorize-service-access.members
}
