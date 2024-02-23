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
  display_name = "Delivery account for ${var.bucket} events in ${local.region}"
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

// Deploy the service into each of our regions.
resource "google_cloud_run_v2_service" "this" {
  project     = var.project_id
  name        = "${var.name}-trampoline"
  description = "A service to deliver events from ${var.bucket} in ${local.region} to the broker."
  location    = local.region

  template {
    scaling {
      min_instance_count = 0
      max_instance_count = 10 // TODO var
    }
    max_instance_request_concurrency = 1000 // TODO var
    execution_environment            = "EXECUTION_ENVIRONMENT_GEN2"

    service_account = google_service_account.this.email
    timeout         = "10s" // TODO var

    containers {
      image = cosign_sign.this.signed_ref

      env {
        name  = "PUBSUB_TOPIC"
        value = var.broker
      }

      resources {
        limits = {
          "cpu"    = "1000m" // TODO var
          "memory" = "512Mi" // TODO var
        }
        cpu_idle = true
      }
    }
    //containers { image = module.otel-collector.image } TODO
  }
}

// Authorize this service account to invoke the private service receiving
// events from this trigger.
module "authorize-delivery" {
  source = "../authorize-private-service"

  project_id = var.project_id
  region     = local.region
  name       = google_cloud_run_v2_service.this.name

  service-account = google_service_account.this.email
}

locals {
  filter-elements = [
    for key, value in var.filter : "attributes.ce-${key}=\"${value}\""
  ]
}

resource "google_pubsub_topic" "dead-letter" {
  name = "${var.name}-dlq-${random_string.suffix.result}"

  message_storage_policy {
    allowed_persistence_regions = [local.region]
  }
}

// Grant the pubsub service account the ability to send to the dead-letter topic.
resource "google_pubsub_topic_iam_binding" "allow-pubsub-to-send-to-dead-letter" {
  topic = google_pubsub_topic.dead-letter.name

  role    = "roles/pubsub.publisher"
  members = ["serviceAccount:${google_project_service_identity.pubsub.email}"]
}

// Configure the subscription to deliver the events matching our filter to this service
// using the above identity to authorize the delivery..
resource "google_pubsub_subscription" "this" {
  depends_on = [google_cloud_run_v2_service.this]

  name  = "${var.name}-${random_string.suffix.result}"
  topic = google_pubsub_topic.internal.id

  // TODO: Tune this and/or make it configurable?
  ack_deadline_seconds = 300

  filter = join(" AND ", local.filter-elements)

  push_config {
    push_endpoint = module.authorize-delivery.uri

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
  subscription = google_pubsub_subscription.this.name

  role    = "roles/pubsub.subscriber"
  members = ["serviceAccount:${google_project_service_identity.pubsub.email}"]
}

// Lookup the GCS service account.
data "google_storage_project_service_account" "gcs_account" {}

// Allow the GCS service account to publish to the internal topic.
resource "google_pubsub_topic_iam_binding" "binding" {
  topic   = google_pubsub_topic.internal.id
  role    = "roles/pubsub.publisher"
  members = ["serviceAccount:${data.google_storage_project_service_account.gcs_account.email_address}"]
}

// Creage the topic to receive the GCS events.
resource "google_pubsub_topic" "internal" {
  name = "${var.name}-internal"

  // TODO: Tune this and/or make it configurable?
  message_retention_duration = "600s"

  message_storage_policy {
    allowed_persistence_regions = [local.region]
  }
}

// Create a notification to the topic.
resource "google_storage_notification" "notification" {
  bucket         = var.bucket
  payload_format = "JSON_API_V1"
  topic          = google_pubsub_topic.internal.id
  event_types    = var.gcs_event_types
  depends_on     = [google_pubsub_topic_iam_binding.binding]
}

// Authorize the trampoline identity to publish events to the topic.
// NOTE: we use binding vs. member because we do not expect anything
// to publish to this topic other than the ingress service.
resource "google_pubsub_topic_iam_binding" "ingress-publishes-events" {
  project = var.project_id
  topic   = var.broker
  role    = "roles/pubsub.publisher"
  members = ["serviceAccount:${google_service_account.this.email}"]
}
