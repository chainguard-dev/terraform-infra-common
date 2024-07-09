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

variable "containers" {
  description = "The containers to run in the service.  Each container will be run in each region."
  type = map(object({
    source = object({
      base_image  = optional(string, "cgr.dev/chainguard/static:latest-glibc")
      working_dir = string
      importpath  = string
    })
    args = optional(list(string), [])
    ports = optional(list(object({
      name           = optional(string, "http1")
      container_port = optional(number, 8080)
    })), [])
    resources = optional(
      object(
        {
          limits = optional(object(
            {
              cpu    = string
              memory = string
            }
          ), null)
          cpu_idle          = optional(bool, true)
          startup_cpu_boost = optional(bool, true)
        }
      ),
      {
        cpu_idle = true
      }
    )
    env = optional(list(object({
      name  = string
      value = optional(string)
      value_source = optional(object({
        secret_key_ref = object({
          secret  = string
          version = string
        })
      }), null)
    })), [])
    regional-env = optional(list(object({
      name  = string
      value = map(string)
    })), [])
    volume_mounts = optional(list(object({
      name       = string
      mount_path = string
    })), [])
  }))
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
  source = "../regional-go-service"

  name            = var.name
  project_id      = var.project_id
  regions         = var.regions
  service_account = google_service_account.sa.email

  egress = "PRIVATE_RANGES_ONLY" // Makes GitHub API calls

  containers = var.containers

  enable_profiler = var.enable_profiler

  notification_channels = var.notification_channels
}

locals {
  combined_filter = merge(
    { "type" : var.github-event }, // Default filter setup
    var.extra_filter               // Merges any additional filters provided
  )

  combined_filter_prefix = merge(
    {},                     // Default empty map
    var.extra_filter_prefix // Merges any provided prefix filters
  )

  combined_filter_has_attributes = concat(
    [],                             // Default empty map
    var.extra_filter_has_attributes // Merges any provided filters for attributes
  )

  combined_filter_not_has_attributes = concat(
    [],                                 // Default empty map
    var.extra_filter_not_has_attributes // Merges any provided filters for not attributes
  )
}

module "cloudevent-trigger" {
  depends_on = [module.service]

  for_each = var.regions
  source   = "../cloudevent-trigger"

  project_id                = var.project_id
  name                      = "bot-trigger-${var.name}"
  broker                    = var.broker[each.key]
  filter                    = local.combined_filter
  filter_prefix             = local.combined_filter_prefix
  filter_has_attributes     = local.combined_filter_has_attributes
  filter_not_has_attributes = local.combined_filter_not_has_attributes

  private-service = {
    region = each.key
    name   = var.name
  }

  notification_channels = var.notification_channels
}

module "dashboard" {
  source = "chainguard-dev/common/infra//modules/dashboard/cloudevent-receiver"

  project_id   = var.project_id
  service_name = var.name

  triggers = {
    (var.name) : {
      subscription_prefix   = "bot-trigger-${var.name}"
      notification_channels = var.notification_channels
    }
  }

  notification_channels = var.notification_channels
}
