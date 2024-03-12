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

data "google_cloud_run_v2_service" "this" {
  for_each = var.service-ingress == "INGRESS_TRAFFIC_ALL" ? var.regions : {}
  project  = var.project_id
  location = each.key
  for_each   = var.service-ingress == "INGRESS_TRAFFIC_ALL" ? var.regions : {}
  project    = var.project_id
  location   = each.key
  name       = var.name
  depends_on = [module.this]
}

output "public-urls" {
  description = "Map of region to public URL for the service, if service-ingress is INGRESS_TRAFFIC_ALL."
  value = var.service-ingress == "INGRESS_TRAFFIC_ALL" ? {
    for r, _ in var.regions : r => data.google_cloud_run_v2_service.this[r].uri
  } : {}
}
