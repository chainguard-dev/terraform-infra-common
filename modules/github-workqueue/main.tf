/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

locals {
  # Service name
  shim_name = "${var.name}-webhook"
}

# Service account for the webhook shim service
resource "google_service_account" "shim" {
  project      = var.project_id
  account_id   = local.shim_name
  display_name = "GitHub webhook service account"
  description  = "Service account for GitHub webhook service that enqueues work"
}

# Authorize the shim to call the workqueue dispatcher in each region
module "shim-calls-workqueue" {
  for_each = var.regions

  source = "../authorize-private-service"

  project_id = var.project_id
  region     = each.key
  name       = var.workqueue.name

  service-account = google_service_account.shim.email
}

# Webhook shim service - public endpoint that receives GitHub webhooks
module "shim" {
  source = "../regional-go-service"

  project_id            = var.project_id
  name                  = local.shim_name
  regions               = var.regions
  service_account       = google_service_account.shim.email
  notification_channels = var.notification_channels

  # Only allow traffic from load balancers (GCLB)
  ingress = "INGRESS_TRAFFIC_INTERNAL_LOAD_BALANCER"

  containers = {
    "shim" = {
      source = {
        working_dir = path.module
        importpath  = "./cmd/shim"
      }
      ports = [{
        container_port = 8080
      }]
      env = concat([
        {
          name = "GITHUB_WEBHOOK_SECRET"
          value_source = {
            secret_key_ref = {
              secret  = module.webhook-secret.secret_id
              version = "latest"
            }
          }
        }
        ], var.resource_filter != "" ? [{
          name  = "RESOURCE_FILTER"
          value = var.resource_filter
      }] : [])
      regional-env = [
        {
          name  = "WORKQUEUE_SERVICE"
          value = { for k, v in module.shim-calls-workqueue : k => v.uri }
        }
      ]
    }
  }
}


# Generate a random webhook secret
resource "random_password" "webhook-secret" {
  length  = 32
  special = true
}

# Use configmap module to manage the webhook secret
module "webhook-secret" {
  source = "../configmap"

  project_id            = var.project_id
  name                  = "${var.name}-webhook-secret"
  data                  = random_password.webhook-secret.result
  service-account       = google_service_account.shim.email
  notification-channels = var.notification_channels
}
