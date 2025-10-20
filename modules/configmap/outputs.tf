output "secret_id" {
  description = "The ID of the secret."
  value       = google_secret_manager_secret.this.id
}

output "secret_version_id" {
  description = "The ID of the secret version."
  value       = google_secret_manager_secret_version.data.id
}

output "version" {
  description = "The secret version."
  value       = google_secret_manager_secret_version.data.version
}
