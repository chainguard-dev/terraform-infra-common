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
  source            = "../sections/grpc"
  title             = "GRPC"
  filter            = []
  service_name      = var.service_name
}

module "resources" {
  source = "../sections/resources"
  title  = "Resources"
  filter = ["resource.type=\"cloud_run_revision\""]
}

module "alerts" {
  for_each = toset(var.alerts)

  source = "../sections/alerts"
  alert  = each.key
  title  = "Alert"
}

module "width" { source = "../sections/width" }

module "layout" {
  source = "../sections/layout"
  sections = concat(
    [for x in var.alerts : module.alerts[x].section],
    [
      module.logs.section,
      module.http.section,
      module.grpc.section,
      module.resources.section,
    ]
  )
}

resource "google_monitoring_dashboard" "dashboard" {
  dashboard_json = jsonencode({
    displayName = "Cloud Run Service: ${var.service_name}"
    labels = merge({
      "service" : ""
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
