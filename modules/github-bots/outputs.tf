output "serviceaccount-id" {
  description = "The ID of the service account for the bot."
  value       = google_service_account.sa.unique_id
}

output "serviceaccount-email" {
  description = "The email of the service account for the bot."
  value       = google_service_account.sa.email
}


