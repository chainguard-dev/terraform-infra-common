/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

module "reconciler" {
  source = "../regional-go-reconciler"

  project_id              = var.project_id
  name                    = var.name
  regions                 = var.regions
  service_account         = var.service_account
  deletion_protection     = var.deletion_protection
  containers              = var.containers
  max-retry               = var.max-retry
  concurrent-work         = var.concurrent-work
  multi_regional_location = var.multi_regional_location
  egress                  = var.egress
  labels                  = var.labels
  squad                   = var.squad
  product                 = var.product
  scaling                 = var.scaling
  volumes                 = var.volumes
  regional-volumes        = var.regional-volumes
  enable_profiler         = var.enable_profiler
  otel_resources          = var.otel_resources
  request_timeout_seconds = var.request_timeout_seconds
  execution_environment   = var.execution_environment
  notification_channels   = var.notification_channels
  workqueue_cpu_idle      = var.workqueue_cpu_idle
  slo                     = var.slo
}

# Authorize the service account to call the receiver in each region
# This is used by both the cron job and push listener
module "authorize-receiver-per-region" {
  for_each = var.regions
  source   = "../authorize-private-service"

  project_id = var.project_id
  region     = each.key
  name       = module.reconciler.receiver.name

  service-account = var.service_account
}
