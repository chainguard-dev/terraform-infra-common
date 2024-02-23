terraform {
  required_providers {
    ko     = { source = "ko-build/ko" }
    cosign = { source = "chainguard-dev/cosign" }
  }
}

resource "random_string" "suffix" {
  length  = 4
  upper   = false
  special = false
}

// A dedicated service account for this subscription.
resource "google_service_account" "this" {
  project = var.project_id

  account_id   = "${var.name}-${random_string.suffix.result}"
  display_name = "Delivery account for ${var.bucket} events"
}

// Lookup the identity of the pubsub service agent.
resource "google_project_service_identity" "pubsub" {
  provider = google-beta
  project  = var.project_id
  service  = "pubsub.googleapis.com"
}

// Authorize Pub/Sub to impersonate the delivery service account to authorize
// deliveries using this service account.
// NOTE: we use binding vs. member because we expect nothing but pubsub to be
// able to assume this identity.
resource "google_service_account_iam_binding" "allow-pubsub-to-mint-tokens" {
  service_account_id = google_service_account.this.name

  role    = "roles/iam.serviceAccountTokenCreator"
  members = ["serviceAccount:${google_project_service_identity.pubsub.email}"]
}

module "audit-serviceaccount" {
  source = "../audit-serviceaccount"

  project_id      = var.project_id
  service-account = google_service_account.this.email

  # The absence of authorized identities here means that
  # nothing is authorized to act as this service account.
  # Note: Cloud Pub/Sub's usage doesn't show up in the
  # audit logs.

  notification_channels = var.notification_channels
}

// Build each of the application images from source.
resource "ko_build" "this" {
  working_dir = path.module
  importpath  = "./cmd/trampoline"
}

// Sign the image.
resource "cosign_sign" "this" {
  image    = ko_build.this.image_ref
  conflict = "REPLACE"
}

module "this" {
  source     = "../regional-go-service"
  project_id = var.project_id
  name       = "${var.name}-trampoline"
  regions    = var.regions

  service_account = google_service_account.this.email
  containers = {
    "recorder" = {
      source = {
        working_dir = path.module
        importpath  = "./cmd/trampoline"
      }
      ports = [{ container_port = 8080 }]
      regional-env = [{
        name  = "INGRESS_URI"
        value = { for k, v in module.trampoline-emits-events : k => v.uri }
      }]
    }
  }

  notification_channels = var.notification_channels
}

// Authorize the Pub/Sub topic to deliver events to the service.
module "authorize-delivery" {
  for_each = var.regions
  source   = "../authorize-private-service"

  project_id = var.project_id
  region     = each.key
  name       = "${var.name}-trampoline"

  service-account = google_service_account.this.email
}

resource "google_pubsub_topic" "dead-letter" {
  name = "${var.name}-dlq-${random_string.suffix.result}"

  message_storage_policy {
    allowed_persistence_regions = keys(var.regions)
  }
}

// Grant the pubsub service account the ability to send to the dead-letter topic.
resource "google_pubsub_topic_iam_binding" "allow-pubsub-to-send-to-dead-letter" {
  for_each = var.regions
  topic    = google_pubsub_topic.dead-letter.name

  role    = "roles/pubsub.publisher"
  members = ["serviceAccount:${google_project_service_identity.pubsub.email}"]
}

// Configure the subscription to deliver the events matching our filter to this service
// using the above identity to authorize the delivery..
resource "google_pubsub_subscription" "this" {
  for_each   = var.regions
  depends_on = [module.this]

  name  = "${var.name}-${random_string.suffix.result}"
  topic = google_pubsub_topic.internal[each.key].id

  // TODO: Tune this and/or make it configurable?
  ack_deadline_seconds = 300

  push_config {
    push_endpoint = module.authorize-delivery[each.key].uri

    // Authenticate requests to this service using tokens minted
    // from the given service account.
    oidc_token {
      service_account_email = google_service_account.this.email
    }

    // Make the body of the push notification the raw Pub/Sub message.
    // Include the Pub/Sub message attributes as HTTP headers.
    // This aligns the shape of the notification with the "binary"
    // Cloud Event delivery form.
    // See: https://cloud.google.com/pubsub/docs/payload-unwrapping
    no_wrapper {
      write_metadata = true
    }
  }

  expiration_policy {
    ttl = "" // This does not expire.
  }

  dead_letter_policy {
    dead_letter_topic     = google_pubsub_topic.dead-letter.id
    max_delivery_attempts = var.max_delivery_attempts
  }
}

// Grant the pubsub service account the ability to Acknowledge messages on this "this" subscription.
resource "google_pubsub_subscription_iam_binding" "allow-pubsub-to-ack" {
  for_each     = var.regions
  subscription = google_pubsub_subscription.this[each.key].name

  role    = "roles/pubsub.subscriber"
  members = ["serviceAccount:${google_project_service_identity.pubsub.email}"]
}

// Lookup the GCS service account.
data "google_storage_project_service_account" "gcs_account" {}

// Allow the GCS service account to publish to the internal topic.
resource "google_pubsub_topic_iam_binding" "binding" {
  for_each = var.regions
  topic    = google_pubsub_topic.internal[each.key].id
  role     = "roles/pubsub.publisher"
  members  = ["serviceAccount:${data.google_storage_project_service_account.gcs_account.email_address}"]
}

// Creage the topic to receive the GCS events.
resource "google_pubsub_topic" "internal" {
  for_each = var.regions
  name     = "${var.name}-internal-${each.key}"

  // TODO: Tune this and/or make it configurable?
  message_retention_duration = "600s"

  message_storage_policy {
    allowed_persistence_regions = [each.key]
  }
}

// Create a notification to the topic.
resource "google_storage_notification" "notification" {
  for_each       = var.regions
  bucket         = var.bucket
  payload_format = "JSON_API_V1"
  topic          = google_pubsub_topic.internal[each.key].id
  event_types    = var.gcs_event_types
  depends_on     = [google_pubsub_topic_iam_binding.binding]
}

// Authorize the trampoline service account to publish events.
module "trampoline-emits-events" {
  for_each = var.regions

  source = "../authorize-private-service"

  project_id = var.project_id
  region     = each.key
  name       = var.ingress.name

  service-account = google_service_account.this.email
}
