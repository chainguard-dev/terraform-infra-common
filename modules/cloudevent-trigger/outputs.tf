output "dead-letter-broker" {
  depends_on  = [google_pubsub_topic.dead-letter]
  value       = google_pubsub_topic.dead-letter.name
  description = "The name of the dead-letter topic, which is used to store events that could not be delivered."
}
