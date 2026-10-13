// The only topic named after the module is the dead-letter topic.
module "topic" {
  source       = "../dashboard/sections/topic"
  title        = "Dead-letter Topic"
  topic_prefix = var.name
}

module "logs" {
  source        = "../dashboard/sections/logs"
  title         = "Service Logs"
  filter        = []
  cloudrun_type = "service"
}

module "http" {
  source       = "../dashboard/sections/http"
  title        = "HTTP"
  filter       = []
  service_name = var.name
}

module "resources" {
  source        = "../dashboard/sections/resources"
  title         = "Resources"
  filter        = []
  cloudrun_name = var.name
  cloudrun_type = "service"

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
    displayName = "Cloud Monitoring Incident Events: ${var.name} in ${var.region}"
    labels = {
      "monitoring" : ""
      "eventing" : ""
    }
    dashboardFilters = [
      {
        # for GCP Cloud Run built-in metrics
        filterType  = "RESOURCE_LABEL"
        stringValue = var.name
        labelKey    = "service_name"
      },
      {
        # for Prometheus user added metrics
        filterType  = "METRIC_LABEL"
        stringValue = var.name
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
