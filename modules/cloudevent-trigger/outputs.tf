output "dead-letter-broker" {
  depends_on  = [google_pubsub_topic.dead-letter]
  value       = google_pubsub_topic.dead-letter.name
  description = "The name of the dead-letter topic, which is used to store events that could not be delivered."
}

output "tag_binding_parents" {
  description = "The parent of every Pub/Sub tag binding, keyed by binding and tag key."
  value = merge(
    { for k, b in google_tags_tag_binding.dead_letter_topic : "dead_letter_topic/${k}" => b.parent },
    { for k, b in google_tags_tag_binding.dead_letter_subscription : "dead_letter_subscription/${k}" => b.parent },
    { for k, b in google_tags_tag_binding.subscription : "subscription/${k}" => b.parent },
  )
}

output "dead-letter-gcs-bucket" {
  depends_on  = [google_storage_bucket.dlq_bucket]
  value       = var.enable_dlq_bucket == false ? null : google_storage_bucket.dlq_bucket[0].name
  description = "The name of the dead-letter GCS bucket, which is used to store events that could not be delivered."
}

output "service_account_email" {
  value       = google_service_account.this.email
  description = "The email of the delivery service account. Each push to the private service carries an OIDC token for it, so a broker ingress receiving this trigger's pushes records it as the events' ce-actor."
}
