resource "random_string" "suffix" {
  length  = 4
  upper   = false
  special = false
}

// A dedicated service account for this subscription.
resource "google_service_account" "this" {
  project = var.project_id

  account_id   = "${var.name}-${random_string.suffix.result}"
  display_name = "Delivery account for ${var.private-service.name} in ${var.private-service.region}."
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

// Authorize this service account to invoke the private service receiving
// events from this trigger.
module "authorize-delivery" {
  source = "../authorize-private-service"

  project_id = var.project_id
  region     = var.private-service.region
  name       = var.private-service.name

  service-account = google_service_account.this.email
}

locals {
  filter-elements = [
    for key, value in var.filter : "attributes.ce-${key}=\"${value}\""
  ]
}

// Configure the subscription to deliver the events matching our filter to this service
// using the above identity to authorize the delivery..
resource "google_pubsub_subscription" "this" {
  name  = "${var.name}-${random_string.suffix.result}"
  topic = var.broker

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
}
