output "secret_id" {
  description = "The ID of the secret."
  value       = google_secret_manager_secret.this.id
}

output "authorized_secret_id" {
  description = "The ID of the secret after its accessor binding is applied."
  value       = google_secret_manager_secret_iam_binding.authorize-service-access.secret_id
}

output "annotations" {
  description = "The annotations set on the secret."
  value       = google_secret_manager_secret.this.annotations
}

output "accessor_members" {
  description = "The members granted roles/secretmanager.secretAccessor on the secret."
  value       = google_secret_manager_secret_iam_binding.authorize-service-access.members
}
