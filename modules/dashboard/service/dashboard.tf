module "errgrp" {
  source       = "../sections/errgrp"
  title        = "Service Error Reporting"
  project_id   = var.project_id
  service_name = var.service_name
}

module "logs" {
  source        = "../sections/logs"
  title         = "Service Logs"
  filter        = []
  cloudrun_type = "service"
}

module "http" {
  source       = "../sections/http"
  title        = "HTTP"
  filter       = []
  service_name = var.service_name
}

module "grpc" {
  source       = "../sections/grpc"
  title        = "GRPC"
  filter       = []
  service_name = var.service_name
}

module "github" {
  source = "../sections/github"
  title  = "GitHub API"
  filter = []
}

module "gorm" {
  source       = "../sections/gorm"
  title        = "GORM"
  filter       = []
  service_name = var.service_name
}

module "resources" {
  source                = "../sections/resources"
  title                 = "Resources"
  filter                = []
  cloudrun_name         = var.service_name
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
      module.errgrp.section,
      module.logs.section,
    ],
    var.sections.http ? [module.http.section] : [],
    var.sections.grpc ? [module.grpc.section] : [],
    var.sections.github ? [module.github.section] : [],
    var.sections.gorm ? [module.gorm.section] : [],
    [module.resources.section],
  )
}

module "dashboard" {
  source = "../"

  object = {
    displayName = "Cloud Run Service: ${var.service_name}"
    labels = merge({
      "service" : ""
    }, var.labels)
    dashboardFilters = [
      {
        # for GCP Cloud Run built-in metrics
        filterType  = "RESOURCE_LABEL"
        stringValue = var.service_name
        labelKey    = "service_name"
      },
      {
        # for Prometheus user added metrics
        filterType  = "METRIC_LABEL"
        stringValue = var.service_name
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
