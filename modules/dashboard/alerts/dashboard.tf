/*
Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

module "alerts" {
  for_each = var.alerts

  source     = "../widgets/alert"
  title      = each.key
  alert_name = each.value
}

module "width" { source = "../sections/width" }

locals {
  columns = 2
  unit    = module.width.size / local.columns

  widgets = [for x in keys(var.alerts) : module.alerts[x].widget]

  tiles = [for x in range(length(local.widgets)) : {
    yPos   = floor(x / 2) * local.unit,
    xPos   = (x % local.columns) * local.unit,
    height = local.unit,
    width  = local.unit,
    widget = local.widgets[x],
  }]
}

module "dashboard" {
  source = "../"

  object = {
    displayName = var.title
    labels      = var.labels

    // https://cloud.google.com/monitoring/api/ref_v3/rest/v1/projects.dashboards#mosaiclayout
    mosaicLayout = {
      columns = module.width.size,
      tiles   = local.tiles,
    }
  }
}
