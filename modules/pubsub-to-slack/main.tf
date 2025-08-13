/*
Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

resource "random_string" "suffix" {
  length  = 4
  upper   = false
  special = false
}

locals {
  default_labels = {
    basename(abspath(path.module)) = var.name
    terraform-module               = basename(abspath(path.module))
  }

  squad_label = var.squad != "" ? {
    squad = var.squad
    team  = var.squad
  } : {}

  product_label = var.product != "" ? {
    product = var.product
  } : {}

  merged_labels = merge(local.default_labels, local.squad_label, local.product_label, var.labels)
}

// Create a service account for the Cloud Run service
resource "google_service_account" "this" {
  project = var.project_id

  account_id   = "${var.name}-${random_string.suffix.result}"
  display_name = "Pub/Sub to Slack bridge service account for ${var.name}"
}

// Grant the service account access to read secrets
resource "google_project_iam_member" "secret_accessor" {
  project = var.project_id
  role    = "roles/secretmanager.secretAccessor"
  member  = "serviceAccount:${google_service_account.this.email}"
}

// Store Slack credentials in Secret Manager
resource "google_secret_manager_secret" "slack_webhook_url" {
  project   = var.project_id
  secret_id = "${var.name}-slack-webhook-url"
  labels    = local.merged_labels

  replication {
    auto {}
  }
}

resource "google_secret_manager_secret_version" "slack_webhook_url" {
  secret      = google_secret_manager_secret.slack_webhook_url.id
  secret_data = var.slack_webhook_url
}

// Create Pub/Sub topic for incoming notifications
resource "google_pubsub_topic" "notifications" {
  name    = var.name
  project = var.project_id
  labels  = local.merged_labels

  message_storage_policy {
    allowed_persistence_regions = [var.region]
  }
}

// Create dead letter topic
resource "google_pubsub_topic" "dead_letter" {
  name    = "${var.name}-dlq"
  project = var.project_id
  labels  = local.merged_labels

  message_storage_policy {
    allowed_persistence_regions = [var.region]
  }
}

// Deploy the Cloud Run service
module "service" {
  source = "../regional-service"

  name       = var.name
  project_id = var.project_id
  squad      = var.squad
  product    = var.product
  labels     = var.labels

  regions = {
    (var.region) = {
      network = var.network
      subnet  = var.subnet
    }
  }

  service_account        = google_service_account.this.email
  notification_channels  = []

  containers = {
    "pubsub-slack-bridge" = {
      image = var.image
      args  = []
      ports = [{
        name           = "http1"
        container_port = 8080
      }]

      env = [{
        name  = "SLACK_WEBHOOK_SECRET"
        value = google_secret_manager_secret.slack_webhook_url.secret_id
        }, {
        name  = "SLACK_CHANNEL"
        value = var.slack_channel
        }, {
        name  = "MESSAGE_TEMPLATE"
        value = var.message_template
        }, {
        name  = "PROJECT_ID"
        value = var.project_id
      }]

      resources = {
        limits = {
          cpu    = var.cpu_limit
          memory = var.memory_limit
        }
      }
    }
  }

  scaling = {
    min_instances                    = var.min_instances
    max_instances                    = var.max_instances
    max_instance_request_concurrency = var.max_concurrency
  }
}

// Get the Cloud Run service URL
locals {
  service_url = module.service.uris[var.region]
}

// Lookup the Pub/Sub service identity
resource "google_project_service_identity" "pubsub" {
  provider = google-beta
  project  = var.project_id
  service  = "pubsub.googleapis.com"
}

// Create push subscription to the Cloud Run service
resource "google_pubsub_subscription" "push" {
  name    = var.name
  project = var.project_id
  topic   = google_pubsub_topic.notifications.name
  labels  = local.merged_labels

  ack_deadline_seconds       = var.ack_deadline_seconds
  message_retention_duration = var.message_retention_duration

  push_config {
    push_endpoint = local.service_url

    oidc_token {
      service_account_email = google_service_account.this.email
    }

    attributes = {
      x-goog-version = "v1"
    }
  }

  dead_letter_policy {
    dead_letter_topic     = google_pubsub_topic.dead_letter.id
    max_delivery_attempts = var.max_delivery_attempts
  }

  retry_policy {
    minimum_backoff = var.min_retry_delay
    maximum_backoff = var.max_retry_delay
  }
}

// Grant Pub/Sub service account permission to invoke the Cloud Run service
resource "google_cloud_run_service_iam_binding" "pubsub_invoker" {
  service  = module.service.names[var.region]
  location = var.region
  project  = var.project_id
  role     = "roles/run.invoker"

  members = [
    "serviceAccount:${google_project_service_identity.pubsub.email}"
  ]
}

// Grant Pub/Sub permission to publish to dead letter topic
resource "google_pubsub_topic_iam_binding" "dead_letter_publisher" {
  topic   = google_pubsub_topic.dead_letter.name
  project = var.project_id
  role    = "roles/pubsub.publisher"

  members = [
    "serviceAccount:${google_project_service_identity.pubsub.email}"
  ]
}
