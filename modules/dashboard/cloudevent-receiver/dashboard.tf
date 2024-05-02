module "logurl" {
  source = "../logurl"

  text    = "Logs Explorer"
  project = var.project_id
  params = {
    "resource.type"                = "cloud_run_revision"
    "resource.labels.service_name" = var.service_name
  }
}

module "markdown" {
  source  = "../sections/markdown"
  title   = "Overview"
  content = <<EOM
  # ${module.logurl.markdown}
  EOM
}

module "subscription" {
  for_each = var.triggers

  source = "../sections/subscription"
  title  = "Events ${each.key}"

  subscription_prefix   = each.value.subscription_prefix
  alert_threshold       = each.value.alert_threshold
  notification_channels = each.value.notification_channels
}

module "errgrp" {
  source       = "../sections/errgrp"
  title        = "Service Error Reporting"
  project_id   = var.project_id
  service_name = var.service_name
}

module "logs" {
  source = "../sections/logs"
  title  = "Service Logs"
  filter = ["resource.type=\"cloud_run_revision\""]
}

module "http" {
  source       = "../sections/http"
  title        = "HTTP"
  filter       = ["resource.type=\"cloud_run_revision\""]
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

module "resources" {
  source        = "../sections/resources"
  title         = "Resources"
  filter        = ["resource.type=\"cloud_run_revision\"", "resource.labels.service_name=\"${var.service_name}\""]
  cloudrun_name = var.service_name

  notification_channels = var.notification_channels
}

module "width" { source = "../sections/width" }

module "layout" {
  source = "../sections/layout"
  sections = concat(
    [module.markdown.section],
    [for key in sort(keys(var.triggers)) : module.subscription[key].section],
    [
      module.errgrp.section,
      module.logs.section,
    ],
    var.sections.http ? [module.http.section] : [],
    var.sections.grpc ? [module.grpc.section] : [],
    var.sections.github ? [module.github.section] : [],
    [module.resources.section],
  )
}

resource "google_monitoring_dashboard" "dashboard" {
  dashboard_json = jsonencode({
    displayName = "Cloud Event Receiver: ${var.service_name}"
    labels = merge({
      "service" : ""
      "eventing" : ""
    }, var.labels)
    dashboardFilters = [{
      filterType  = "RESOURCE_LABEL"
      stringValue = var.service_name
      labelKey    = "service_name"
    }]

    // https://cloud.google.com/monitoring/api/ref_v3/rest/v1/projects.dashboards#mosaiclayout
    mosaicLayout = {
      columns = module.width.size
      tiles   = module.layout.tiles,
    }
  })
}
