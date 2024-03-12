resource "random_string" "service-suffix" {
  length  = 4
  upper   = false
  special = false
}

// A dedicated service account for the trampoline service.
resource "google_service_account" "service" {
  project = var.project_id

  account_id   = "${var.name}-${random_string.service-suffix.result}"
  display_name = "Service account for GitHub events trampoline service"
}

module "webhook-secret" {
  source = "../secret"

  project_id = var.project_id
  name       = "${var.name}-webhook-secret"

  service-account  = google_service_account.service.email
  authorized-adder = var.secret_version_adder

  notification-channels = var.notification_channels
}

// If set, populate the webhook secret latest version.
resource "google_secret_manager_secret_version" "webhook-secret" {
  count = var.webhook-secret != "" ? 1 : 0

  secret      = module.webhook-secret.secret_id
  secret_data = var.webhook-secret
}

module "this" {
  source     = "../regional-go-service"
  project_id = var.project_id
  name       = var.name
  regions    = var.regions

  ingress = var.service-ingress

  service_account = google_service_account.service.email
  containers = {
    "trampoline" = {
      source = {
        working_dir = path.module
        importpath  = "./cmd/trampoline"
      }
      ports = [{ container_port = 8080 }]
      env = [{
        name = "WEBHOOK_SECRET"
        value_source = {
          secret_key_ref = {
            secret  = module.webhook-secret.secret_id
            version = "latest"
          }
        }
      }]
      regional-env = [{
        name  = "EVENT_INGRESS_URI"
        value = { for k, v in module.trampoline-emits-events : k => v.uri }
      }]
    }
  }

  notification_channels = var.notification_channels
}

// Authorize the trampoline service account to publish events on the broker.
module "trampoline-emits-events" {
  for_each = var.regions
  source   = "../authorize-private-service"

  project_id = var.project_id
  region     = each.key
  name       = var.ingress.name

  service-account = google_service_account.service.email
}
