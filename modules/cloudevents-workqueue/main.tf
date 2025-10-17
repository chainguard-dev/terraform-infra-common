terraform {
  required_providers {
    cosign = { source = "chainguard-dev/cosign" }
    google = { source = "hashicorp/google" }
    random = { source = "hashicorp/random" }
  }
}

// Create a service account for the service
resource "google_service_account" "subscriber" {
  project = var.project_id

  account_id   = "${var.name}-sub"
  display_name = "CloudEvents to Workqueue Subscriber"
  description  = "Service account for ${var.name} CloudEvents subscriber"
}

// Deploy the subscriber service
module "subscriber" {
  source = "../regional-go-service"

  project_id = var.project_id
  name       = var.name
  regions    = var.regions

  service_account = google_service_account.subscriber.email

  notification_channels = var.notification_channels
  deletion_protection   = var.deletion_protection

  squad = var.squad

  containers = {
    "subscriber" = {
      source = {
        importpath  = "./cmd/subscriber"
        working_dir = path.module
      }
      ports = [{
        container_port = 8080
      }]
      env = [{
        name  = "EXTENSION_KEY"
        value = var.extension_key
      }]
      regional-env = [
        {
          name  = "WORKQUEUE_SERVICE"
          value = { for k, v in module.subscriber-calls-workqueue : k => v.uri }
        }
      ]
    }
  }
}

// Authorize the subscriber to call the workqueue in each region
module "subscriber-calls-workqueue" {
  for_each = var.regions

  source = "../authorize-private-service"

  project_id      = var.project_id
  region          = each.key
  name            = var.workqueue.name
  service-account = google_service_account.subscriber.email
}

// Create a subscription to the broker with filters for the specified event types
// We need a trigger for each region and each filter
module "trigger" {
  for_each = {
    for pair in setproduct(keys(var.regions), range(length(var.filters))) :
    "${pair[0]}-${pair[1]}" => {
      region = pair[0]
      filter = var.filters[pair[1]]
      index  = pair[1]
      broker = var.broker[pair[0]]
    }
  }

  source = "../cloudevent-trigger"

  project_id = var.project_id
  name       = "${var.name}-${each.value.region}-${each.value.index}"
  broker     = each.value.broker

  private-service = {
    name   = var.name
    region = each.value.region
  }

  // Pass the filter and ensure extension key exists
  filter                = each.value.filter
  filter_has_attributes = [var.extension_key]

  notification_channels = var.notification_channels

  max_delivery_attempts = var.max_delivery_attempts
  minimum_backoff       = var.minimum_backoff
  maximum_backoff       = var.maximum_backoff
  ack_deadline_seconds  = var.ack_deadline_seconds

  team = var.squad

  depends_on = [module.subscriber]
}
