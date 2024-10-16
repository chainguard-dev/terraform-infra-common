output "serviceaccount-id" {
  description = "The ID of the service account for the bot."
  value       = var.service_account_email == "" ? google_service_account.sa[0].unique_id : ""
}

output "serviceaccount-email" {
  description = "The email of the service account for the bot."
  value       = var.service_account_email == "" ? google_service_account.sa[0].email : var.service_account_email
}


