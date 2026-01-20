/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

terraform {
  required_providers {
    ko     = { source = "ko-build/ko" }
    cosign = { source = "chainguard-dev/cosign" }
  }
}

locals {
  # Floor division for all shards
  base_concurrency = floor(var.concurrent-work / var.shards)

  # Remainder distributed across first N shards (1 extra each)
  remainder = var.concurrent-work - (local.base_concurrency * var.shards)

  # Flatten (region, shard) pairs for authorize-private-service calls
  auth_pairs = flatten([
    for region in keys(var.regions) : [
      for shard in range(var.shards) : {
        key    = "${region}-${shard}"
        region = region
        shard  = tostring(shard)
      }
    ]
  ])
}

# Instantiate N workqueue modules
module "workqueue" {
  for_each = toset([for i in range(var.shards) : tostring(i)])

  source = "../"

  project_id = var.project_id
  name       = "${var.name}-${each.key}"
  regions    = var.regions

  reconciler-service = var.reconciler-service
  # First `remainder` shards get base+1, rest get base
  concurrent-work             = tonumber(each.key) < local.remainder ? local.base_concurrency + 1 : local.base_concurrency
  batch-size                  = var.batch-size
  max-retry                   = var.max-retry
  enable_dead_letter_alerting = var.enable_dead_letter_alerting

  team                    = var.team
  product                 = var.product
  deletion_protection     = var.deletion_protection
  notification_channels   = var.notification_channels
  labels                  = var.labels
  multi_regional_location = var.multi_regional_location
}

# Service account for hyperqueue router
resource "google_service_account" "hyperqueue" {
  project    = var.project_id
  account_id = "${var.name}-hq"
}

# Authorize hyperqueue to call each workqueue's receiver
module "hyperqueue-calls-receiver" {
  for_each = { for pair in local.auth_pairs : pair.key => pair }

  source = "../../authorize-private-service"

  project_id = var.project_id
  region     = each.value.region
  name       = module.workqueue[each.value.shard].receiver.name

  service-account = google_service_account.hyperqueue.email
}

# Hyperqueue service using regional-go-service
module "hyperqueue-service" {
  source     = "../../regional-go-service"
  project_id = var.project_id
  name       = "${var.name}-hq"
  regions    = var.regions
  labels     = merge({ "service" : "workqueue-hyperqueue" }, var.labels)
  team       = var.team
  product    = var.product

  deletion_protection = var.deletion_protection
  service_account     = google_service_account.hyperqueue.email

  containers = {
    "hyperqueue" = {
      source = {
        working_dir = path.module
        importpath  = "github.com/chainguard-dev/terraform-infra-common/modules/workqueue/hyperqueue/cmd/hyperqueue"
      }
      ports = [{ container_port = 8080 }]
      regional-env = [
        {
          name = "SHARD_URLS"
          value = {
            for region in keys(var.regions) : region => join(",", [
              for shard in range(var.shards) : module.hyperqueue-calls-receiver["${region}-${shard}"].uri
            ])
          }
        },
      ]
    }
  }

  notification_channels = var.notification_channels
}
