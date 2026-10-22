output "secret_id" {
  description = "The ID of the secret."
  value       = google_secret_manager_secret.this.id
}

output "authorized_secret_id" {
  description = "The ID of the secret after its accessor binding is applied."
  value       = google_secret_manager_secret_iam_binding.authorize-service-access.secret_id
}
