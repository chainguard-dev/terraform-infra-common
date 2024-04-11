variable "name" {
  description = "The name of the bot."
  type        = string
}

variable "project_id" {
  description = "Project ID to create resources in."
  type        = string
}

variable "regions" {
  description = "A map from region names to a network and subnetwork."
  type = map(object({
    network = string
    subnet  = string
  }))
}

variable "broker" {
  description = "A map from each of the input region names to the name of the Broker topic in that region."
  type        = map(string)
}

variable "source_code" {
  description = "The source code for the bot."
  type = object({
    working_dir = string
    importpath  = string
  })
}

variable "github-event" {
  description = "The GitHub event type to subscribe to."
  type        = string
}

variable "notification_channels" {
  description = "List of notification channels to alert."
  type        = list(string)
}

output "serviceaccount-id" {
  description = "The ID of the service account for the bot."
  value       = google_service_account.sa.unique_id
}

output "serviceaccount-email" {
  description = "The ID of the service account for the bot."
  value       = google_service_account.sa.email
}

resource "google_service_account" "sa" {
  account_id   = "bot-${var.name}"
  display_name = "Service Account for ${var.name}"
}

module "service" {
  source = "chainguard-dev/common/infra//modules/regional-go-service"

  name            = var.name
  project_id      = var.project_id
  regions         = var.regions
  service_account = google_service_account.sa.email

  egress = "PRIVATE_RANGES_ONLY" // Makes GitHub API calls

  containers = {
    "${var.name}" = {
      source = var.source_code
      ports  = [{ container_port = 8080 }]
    }
  }

  notification_channels = var.notification_channels
}

module "cloudevent-trigger" {
  depends_on = [module.service]

  for_each = var.regions
  source   = "chainguard-dev/common/infra//modules/cloudevent-trigger"

  project_id = var.project_id
  name       = "bot-trigger"
  broker     = var.broker[each.key]
  filter     = { "type" : var.github-event }

  private-service = {
    region = each.key
    name   = var.name
  }

  notification_channels = var.notification_channels
}
