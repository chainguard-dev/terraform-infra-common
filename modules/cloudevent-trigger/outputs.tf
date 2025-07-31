output "dead-letter-broker" {
  depends_on  = [google_pubsub_topic.dead-letter]
  value       = google_pubsub_topic.dead-letter.name
  description = "The name of the dead-letter topic, which is used to store events that could not be delivered."
}

output "dead-letter-gcs-bucket" {
  depends_on  = [google_storage_bucket.dlq_bucket]
  value       = var.enable_dlq_bucket == false ? null : google_storage_bucket.dlq_bucket[0].name
  description = "The name of the dead-letter GCS bucket, which is used to store events that could not be delivered."
}
