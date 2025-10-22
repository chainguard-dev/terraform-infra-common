/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

module "push-listener" {
  source = "../regional-go-service"

  name       = "${var.name}-push"
  project_id = var.project_id
  regions    = var.regions

  service_account = var.service_account

  containers = {
    push-listener = {
      source = {
        working_dir = path.module
        importpath  = "./cmd/push"
      }
      ports = [{
        container_port = 8080
      }]
      env = [{
        name  = "PATH_PATTERNS"
        value = jsonencode(var.path_patterns)
        }, {
        name  = "OCTO_IDENTITY"
        value = var.octo_sts_identity
      }]
      regional-env = [{
        name  = "WORKQUEUE_ADDR"
        value = { for region, auth in module.authorize-receiver-per-region : region => auth.uri }
      }]
    }
  }

  egress = "PRIVATE_RANGES_ONLY"

  deletion_protection   = var.deletion_protection
  notification_channels = var.notification_channels
  labels                = var.labels
  product               = var.product
  team                  = var.team
}

# Subscribe to push events in each region
module "push-subscription" {
  for_each = var.paused ? {} : var.regions
  source   = "../cloudevent-trigger"

  name   = "${var.name}-push"
  broker = var.broker[each.key]
  filter = {
    type    = "dev.chainguard.github.push"
    subject = "${var.github_owner}/${var.github_repo}"
  }

  private-service = {
    region = each.key
    name   = "${var.name}-push"
  }

  project_id = var.project_id

  product = var.product
  team    = var.team

  notification_channels = var.notification_channels

  depends_on = [module.push-listener]
}
