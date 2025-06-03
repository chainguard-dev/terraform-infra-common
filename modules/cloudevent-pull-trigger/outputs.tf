output "dead-letter-broker" {
  depends_on  = [google_pubsub_topic.dead-letter]
  value       = google_pubsub_topic.dead-letter.name
  description = "The name of the dead-letter topic, which is used to store events that could not be delivered."
}

output "pull-subscription" {
  depends_on  = [google_pubsub_subscription.this]
  value       = google_pubsub_subscription.this.name
  description = "The name of the pull subscription, which is used to receive events from the broker."
}
