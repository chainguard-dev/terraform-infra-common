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

// Stand up the workqueue infrastructure
module "workqueue" {
  source = "../workqueue"

  project_id = var.project_id
  name       = "${var.name}-wq"
  regions    = var.regions

  concurrent-work = var.concurrent-work
  max-retry       = var.max-retry

  reconciler-service = {
    name = "${var.name}-rec"
  }

  team                  = var.team
  product               = var.product
  deletion_protection   = var.deletion_protection
  notification_channels = var.notification_channels
  labels                = var.labels

  multi_regional_location = var.multi_regional_location
  cpu_idle                = var.workqueue_cpu_idle

  depends_on = [module.reconciler]
}

// Stand up the reconciler service
module "reconciler" {
  source = "../regional-go-service"

  project_id = var.project_id
  name       = "${var.name}-rec"
  regions    = var.regions
  ingress    = "INGRESS_TRAFFIC_INTERNAL_ONLY"
  egress     = var.egress

  deletion_protection = var.deletion_protection

  service_account = var.service_account
  containers      = var.containers

  labels           = var.labels
  team             = var.team
  squad            = var.squad
  product          = var.product
  scaling          = var.scaling
  volumes          = var.volumes
  regional-volumes = var.regional-volumes
  enable_profiler  = var.enable_profiler
  otel_resources   = var.otel_resources

  request_timeout_seconds = var.request_timeout_seconds
  execution_environment   = var.execution_environment

  slo = var.slo

  notification_channels = var.notification_channels
}
