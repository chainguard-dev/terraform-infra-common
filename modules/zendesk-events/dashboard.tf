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

// Redaction health. zendesk_redact_fail_closed_total is the load-bearing privacy
// signal: every increment means redact.Body degraded to dropping fields (regex
// pass produced invalid JSON, or a non-object body), so any sustained rate is
// meant to be alerted on.
module "redact_fail_closed" {
  source = "../dashboard/widgets/xy"
  title  = "Redaction fail-closed rate (by reason)"
  filter = [
    "metric.type=\"prometheus.googleapis.com/zendesk_redact_fail_closed_total/counter\"",
    "resource.type=\"prometheus_target\"",
  ]
  group_by_fields = ["metric.label.\"reason\""]
  plot_type       = "STACKED_BAR"
  primary_align   = "ALIGN_RATE"
  primary_reduce  = "REDUCE_SUM"
}

module "redaction" {
  source = "../dashboard/sections/collapsible"
  title  = "Redaction"
  tiles = [{
    yPos   = 0
    xPos   = 0
    height = module.width.size / 2
    width  = module.width.size / 2
    widget = module.redact_fail_closed.widget
  }]
}

module "layout" {
  source = "../dashboard/sections/layout"
  sections = [
    module.redaction.section,
    module.logs.section,
    module.http.section,
    module.resources.section,
  ]
}

module "dashboard" {
  source = "../dashboard"

  object = {
    displayName = "Zendesk Webhook Events"
    labels = {
      "zendesk" : ""
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
