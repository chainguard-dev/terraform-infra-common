resource "random_string" "suffix" {
  length  = 4
  upper   = false
  special = false
}

// Lookup the identity of the pubsub service agent.
resource "google_project_service_identity" "pubsub" {
  provider = google-beta
  project  = var.project_id
  service  = "pubsub.googleapis.com"
}

locals {
  // See https://cloud.google.com/pubsub/docs/subscription-message-filter#filtering_syntax
  filter-elements = concat(
    [for key, value in var.filter : "attributes.ce-${key}=\"${value}\""],
    [for key, value in var.filter_prefix : "hasPrefix(attributes.ce-${key}, \"${value}\")"],
    [for key in var.filter_has_attributes : "attributes:ce-${key}"],
    [for key in var.filter_not_has_attributes : "NOT attributes:ce-${key}"],
  )
}

resource "google_pubsub_topic" "dead-letter" {
  name = "${var.name}-dlq-${random_string.suffix.result}"

  labels = var.team == "" ? {} : {
    team = var.team
  }

  message_storage_policy {
    allowed_persistence_regions = var.allowed_persistence_regions
  }
}

// Grant the pubsub service account the ability to send to the dead-letter topic.
resource "google_pubsub_topic_iam_binding" "allow-pubsub-to-send-to-dead-letter" {
  topic = google_pubsub_topic.dead-letter.name

  role    = "roles/pubsub.publisher"
  members = ["serviceAccount:${google_project_service_identity.pubsub.email}"]
}

// Create a subscription to the dead-letter topic so dead-lettered messages
// are retained. This also allows us to alerts based on better metrics
// like the age or count of dead-lettered messages.
resource "google_pubsub_subscription" "dead-letter-pull-sub" {
  name                       = google_pubsub_topic.dead-letter.name
  topic                      = google_pubsub_topic.dead-letter.name
  message_retention_duration = "86400s"

  expiration_policy {
    ttl = "86400s"
  }

  enable_message_ordering = true
}

// Configure the subscription to the broker topic with the appropriate filter.
resource "google_pubsub_subscription" "this" {
  name  = "${var.name}-${random_string.suffix.result}"
  topic = var.broker

  ack_deadline_seconds = var.ack_deadline_seconds

  filter = var.raw_filter == "" ? join(" AND ", local.filter-elements) : var.raw_filter

  expiration_policy {
    ttl = "" // This does not expire.
  }

  dead_letter_policy {
    dead_letter_topic     = google_pubsub_topic.dead-letter.id
    max_delivery_attempts = var.max_delivery_attempts
  }

  retry_policy {
    minimum_backoff = "${var.minimum_backoff}s"
    maximum_backoff = "${var.maximum_backoff}s"
  }
}
