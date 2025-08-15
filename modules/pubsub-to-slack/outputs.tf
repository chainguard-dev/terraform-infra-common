/*
Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

output "topic_name" {
  description = "The name of the Pub/Sub topic that other services can publish to"
  value       = google_pubsub_topic.notifications.name
}

output "topic_id" {
  description = "The full resource ID of the Pub/Sub topic"
  value       = google_pubsub_topic.notifications.id
}

output "subscription_name" {
  description = "The name of the Pub/Sub subscription"
  value       = google_pubsub_subscription.push.name
}

output "dead_letter_topic_name" {
  description = "The name of the dead letter topic"
  value       = google_pubsub_topic.dead_letter.name
}

output "service_account_email" {
  description = "The email of the service account used by the Cloud Run service"
  value       = google_service_account.this.email
}


output "service_url" {
  description = "Cloud Run service URL"
  value       = module.service.uris[var.region]
}
