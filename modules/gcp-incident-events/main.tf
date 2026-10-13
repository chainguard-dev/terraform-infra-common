terraform {
  required_version = ">= 1.9"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = ">= 7.21.0"
    }
    google-beta = {
      source  = "hashicorp/google-beta"
      version = ">= 7.21.0"
    }
    random = {
      source  = "hashicorp/random"
      version = ">= 3.9.0"
    }
  }
}

locals {
  default_labels = {
    basename(abspath(path.module)) = var.name
    terraform-module               = basename(abspath(path.module))
  }

  squad_label = {
    squad = var.team
  }

  merged_labels = merge(local.default_labels, local.squad_label, var.labels)
}

resource "random_string" "service-suffix" {
  length  = 4
  upper   = false
  special = false
}

// A dedicated service account for the trampoline service.
resource "google_service_account" "service" {
  project = var.project_id

  account_id   = "${var.name}-${random_string.service-suffix.result}"
  display_name = "Service account for Cloud Monitoring incident trampoline service"
}

resource "random_string" "delivery-suffix" {
  length  = 4
  upper   = false
  special = false
}

// A dedicated service account for this subscription.
resource "google_service_account" "delivery" {
  project = var.project_id

  account_id   = "${var.name}-${random_string.delivery-suffix.result}"
  display_name = "Delivery account for Cloud Monitoring incident events"
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
  service_account_id = google_service_account.delivery.name

  role    = "roles/iam.serviceAccountTokenCreator"
  members = ["serviceAccount:${google_project_service_identity.pubsub.email}"]
}

// A push subscription has a single endpoint, so the trampoline runs in one
// region.
module "this" {
  source             = "../regional-go-service"
  observability_role = var.observability_role
  project_id         = var.project_id
  name               = var.name
  regions = {
    (var.region) : var.regions[var.region]
  }

  team                = var.team
  product             = var.product
  deletion_protection = var.deletion_protection
  service_account     = google_service_account.service.email
  containers = {
    "trampoline" = {
      source = {
        working_dir = path.module
        importpath  = "./cmd/trampoline"
      }
      ports = [{ container_port = 8080 }]
      env = [{
        name  = "EVENT_INGRESS_URI"
        value = module.trampoline-emits-events.uri
      }]
    }
  }

  enable_profiler = var.enable_profiler

  notification_channels = var.notification_channels

  resource_manager_tags = var.resource_manager_tags
}

// Authorize the Pub/Sub subscription to deliver notifications to the service.
module "authorize-delivery" {
  source = "../authorize-private-service"

  project_id = var.project_id
  region     = var.region
  name       = var.name

  service-account = google_service_account.delivery.email
}

resource "google_pubsub_topic" "dead-letter" {
  name = "${var.name}-dlq-${random_string.delivery-suffix.result}"
  // The alerting module's per-team DLQ alert matches dead-letter topics on
  // the team label. Keep team until that alert switches to squad.
  labels = merge({ team = var.team }, local.merged_labels)

  message_storage_policy {
    allowed_persistence_regions = [var.region]
  }
}

// Create a subscription to the dead-letter topic so dead-lettered messages
// are retained, which also lets alerts use their age and count.
resource "google_pubsub_subscription" "dead-letter-pull-sub" {
  name                       = google_pubsub_topic.dead-letter.name
  topic                      = google_pubsub_topic.dead-letter.name
  labels                     = local.merged_labels
  message_retention_duration = "86400s"

  // Never expire this subscription. Pub/Sub deletes subscriptions with no
  // subscriber activity after the expiration TTL, and nothing pulls from
  // this subscription in steady state, so any finite TTL means it silently
  // disappears about a day after creation. Messages published to a topic
  // with no subscriptions are dropped, so once that happens dead-lettered
  // incidents are lost with no retention and no replay path.
  expiration_policy {
    ttl = ""
  }

  enable_message_ordering = true
}

// Grant the pubsub service account the ability to send to the dead-letter topic.
resource "google_pubsub_topic_iam_binding" "allow-pubsub-to-send-to-dead-letter" {
  topic = google_pubsub_topic.dead-letter.name

  role    = "roles/pubsub.publisher"
  members = ["serviceAccount:${google_project_service_identity.pubsub.email}"]
}

// Deliver every notification on the incident topic to the trampoline, using
// the delivery identity to authorize the push.
resource "google_pubsub_subscription" "this" {
  // Ensure the trampoline service is ready and Pub/Sub can mint OIDC tokens
  // for the delivery SA before creating the push subscription.
  depends_on = [
    module.this,
    google_service_account_iam_binding.allow-pubsub-to-mint-tokens,
  ]

  name   = "${var.name}-${random_string.delivery-suffix.result}"
  topic  = var.topic
  labels = local.merged_labels

  ack_deadline_seconds = 300

  push_config {
    push_endpoint = module.authorize-delivery.uri

    // Authenticate requests to this service using tokens minted
    // from the given service account.
    oidc_token {
      service_account_email = google_service_account.delivery.email
    }

    // The trampoline forwards the notification body as the event data and
    // reads the message ID from the metadata headers.
    // See: https://cloud.google.com/pubsub/docs/payload-unwrapping
    no_wrapper {
      write_metadata = true
    }
  }

  // Back off so a short broker outage does not use up every delivery
  // attempt before the broker recovers.
  retry_policy {
    minimum_backoff = "10s"
    maximum_backoff = "600s"
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

// Authorize the trampoline service account to publish events.
module "trampoline-emits-events" {
  source = "../authorize-private-service"

  project_id = var.project_id
  region     = var.region
  name       = var.ingress.name

  service-account = google_service_account.service.email
}

output "recorder-schemas" {
  value = {
    "dev.chainguard.gcp.monitoring.incident" : {
      schema = file("${path.module}/schemas/incident.schema.json")
    }
  }
}
