resource "random_string" "suffix" {
  length  = 4
  upper   = false
  special = false
}

// If method is gcs, GCP subscription will write events directly to a GCS bucket.
resource "google_storage_bucket_iam_binding" "broker-writes-to-gcs-buckets" {
  for_each = var.method == "gcs" ? var.regions : {}

  bucket  = google_storage_bucket.recorder[each.key].name
  role    = "roles/storage.admin"
  members = ["serviceAccount:${google_project_service_identity.pubsub.email}"]
}

// Lookup the identity of the pubsub service agent.
resource "google_project_service_identity" "pubsub" {
  provider = google-beta
  project  = var.project_id
  service  = "pubsub.googleapis.com"
}

resource "google_pubsub_topic" "dead-letter" {
  for_each = var.method == "gcs" ? local.regional-types : {}

  name = "${var.name}-dlq-${substr(md5(each.key), 0, 6)}"

  message_storage_policy {
    allowed_persistence_regions = [each.value.region]
  }
}

// Grant the pubsub service account the ability to send to the dead-letter topic.
resource "google_pubsub_topic_iam_binding" "allow-pubsub-to-send-to-dead-letter" {
  for_each = var.method == "gcs" ? local.regional-types : {}
  topic    = google_pubsub_topic.dead-letter[each.key].name

  role    = "roles/pubsub.publisher"
  members = ["serviceAccount:${google_project_service_identity.pubsub.email}"]
}

// Configure the subscription to deliver the events matching our filter to this service
// using the above identity to authorize the delivery..
resource "google_pubsub_subscription" "this" {
  for_each = var.method == "gcs" ? local.regional-types : {}
  name     = "${var.name}-${substr(md5(each.key), 0, 6)}"

  topic = var.broker[each.value.region]

  ack_deadline_seconds = var.ack_deadline_seconds

  filter = "attributes.ce-type=\"${each.value.type}\""

  cloud_storage_config {
    bucket          = google_storage_bucket.recorder[each.value.region].name
    filename_prefix = "${each.value.type}/"
    max_bytes       = var.cloud_storage_config_max_bytes
    max_duration    = "${var.cloud_storage_config_max_duration}s"
  }

  expiration_policy {
    ttl = "" // This does not expire.
  }

  dead_letter_policy {
    dead_letter_topic     = google_pubsub_topic.dead-letter[each.key].id
    max_delivery_attempts = var.max_delivery_attempts
  }

  retry_policy {
    minimum_backoff = "${var.minimum_backoff}s"
    maximum_backoff = "${var.maximum_backoff}s"
  }
}

// Grant the pubsub service account the ability to Acknowledge messages on this "this" subscription.
resource "google_pubsub_subscription_iam_binding" "allow-pubsub-to-ack" {
  for_each     = var.method == "gcs" ? local.regional-types : {}
  subscription = google_pubsub_subscription.this[each.key].name

  role    = "roles/pubsub.subscriber"
  members = ["serviceAccount:${google_project_service_identity.pubsub.email}"]
}
