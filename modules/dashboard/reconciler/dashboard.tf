/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

locals {
  service_name   = var.service_name != "" ? var.service_name : "${var.name}-rec"
  workqueue_name = var.workqueue_name != "" ? var.workqueue_name : "${var.name}-wq"
}

// Workqueue metrics section
module "workqueue-state" {
  source = "../sections/workqueue"

  title           = "Workqueue State"
  service_name    = local.workqueue_name
  max_retry       = var.max_retry
  concurrent_work = var.concurrent_work
  filter          = []
  collapsed       = false
}

// Reconciler service sections
module "errgrp" {
  source       = "../sections/errgrp"
  title        = "Reconciler Error Reporting"
  project_id   = var.project_id
  service_name = local.service_name
  collapsed    = true
}

module "reconciler-logs" {
  source        = "../sections/logs"
  title         = "Reconciler Logs"
  filter        = ["resource.labels.service_name=\"${local.service_name}\""]
  cloudrun_type = "service"
}

module "grpc" {
  source       = "../sections/grpc"
  title        = "GRPC"
  filter       = []
  service_name = local.service_name
}

module "github" {
  source = "../sections/github"
  title  = "GitHub API"
  filter = []
}

module "resources" {
  source                = "../sections/resources"
  title                 = "Reconciler Resources"
  filter                = []
  cloudrun_name         = local.service_name
  cloudrun_type         = "service"
  notification_channels = var.notification_channels
}

module "alerts" {
  for_each = var.alerts

  source = "../sections/alerts"
  alert  = each.value
  title  = "Alert: ${each.key}"
}

module "width" { source = "../sections/width" }

module "layout" {
  source = "../sections/layout"
  sections = concat(
    [for x in keys(var.alerts) : module.alerts[x].section],
    [
      module.workqueue-state.section,
      module.errgrp.section,
      module.reconciler-logs.section,
      module.grpc.section,
    ],
    var.sections.github ? [module.github.section] : [],
    [module.resources.section],
  )
}

module "dashboard" {
  source = "../"

  object = {
    displayName = "Reconciler: ${var.name}"
    labels = merge({
      "service" : ""
      "reconciler" : ""
    }, var.labels)
    dashboardFilters = [
      {
        # for GCP Cloud Run built-in metrics
        filterType  = "RESOURCE_LABEL"
        stringValue = local.service_name
        labelKey    = "service_name"
      },
      {
        # for Prometheus user added metrics
        filterType  = "METRIC_LABEL"
        stringValue = local.service_name
        labelKey    = "service_name"
      },
    ]

    // https://cloud.google.com/monitoring/api/ref_v3/rest/v1/projects.dashboards#mosaiclayout
    mosaicLayout = {
      columns = module.width.size
      tiles   = module.layout.tiles,
    }
  }
}
