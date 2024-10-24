module "topic" {
  source       = "../dashboard/sections/topic"
  title        = "Notification Topic"
  topic_prefix = var.name
}

module "logs" {
  source = "../dashboard/sections/logs"
  title  = "Service Logs"
  filter = ["resource.type=\"cloud_run_revision\""]
}

module "http" {
  source       = "../dashboard/sections/http"
  title        = "HTTP"
  filter       = ["resource.type=\"cloud_run_revision\""]
  service_name = var.name
}

module "resources" {
  source        = "../dashboard/sections/resources"
  title         = "Resources"
  filter        = ["resource.type=\"cloud_run_revision\"", "resource.labels.service_name=\"${var.name}\""]
  cloudrun_name = var.name

  notification_channels = var.notification_channels
}

module "width" { source = "../dashboard/sections/width" }

module "layout" {
  source = "../dashboard/sections/layout"
  sections = [
    module.topic.section,
    module.logs.section,
    module.http.section,
    module.resources.section,
  ]
}

module "dashboard" {
  source = "../dashboard"

  object = {
    displayName = "GCS Bucket Events: ${var.bucket} in ${local.region}"
    labels = {
      "storage" : ""
      "eventing" : ""
    }
    dashboardFilters = [{
      filterType  = "RESOURCE_LABEL"
      stringValue = var.name
      labelKey    = "service_name"
    }]

    // https://cloud.google.com/monitoring/api/ref_v3/rest/v1/projects.dashboards#mosaiclayout
    mosaicLayout = {
      columns = module.width.size
      tiles   = module.layout.tiles,
    }
  }
}

moved {
  from = google_monitoring_dashboard.dashboard
  to = module.dashboard.google_monitoring_dashboard.dashboard
}
