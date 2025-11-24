/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

locals {
  # Construct cron schedule from resync period:
  # If < 24 hours, use "0 */N * * *" (every N hours)
  # If >= 24 hours, use "0 0 */D * * *" (every D days)
  cron_schedule = var.resync_period_hours < 24 ? "0 */${var.resync_period_hours} * * *" : "0 0 */${floor(var.resync_period_hours / 24)} * * *"
  # Period in minutes for time bucketing
  period_minutes = var.resync_period_hours * 60
}

module "cron" {
  source = "../cron"

  name       = "${var.name}-enq"
  project_id = var.project_id
  region     = var.primary-region

  importpath  = "./cmd/resync"
  working_dir = path.module

  service_account = var.service_account
  schedule        = local.cron_schedule
  paused          = var.paused

  env = {
    GITHUB_OWNER      = var.github_owner
    GITHUB_REPO       = var.github_repo
    OCTO_STS_IDENTITY = var.octo_sts_identity
    WORKQUEUE_ADDR    = module.authorize-receiver-per-region[var.primary-region].uri
    PATH_PATTERNS     = jsonencode(var.path_patterns)
    PERIOD_MINUTES    = tostring(local.period_minutes)
  }

  vpc_access = {
    network_interfaces = [{
      network    = var.regions[var.primary-region].network
      subnetwork = var.regions[var.primary-region].subnet
    }]
    egress = "PRIVATE_RANGES_ONLY"
  }

  notification_channels = var.notification_channels
  deletion_protection   = var.deletion_protection
  labels                = var.labels
  team                  = var.team
  product               = var.product
}
