module "subscription" {
  for_each = var.triggers

  source = "../sections/subscription"
  title  = "Events ${each.key}"

  subscription_prefix = each.value
}

module "logs" {
  source = "../sections/logs"
  title  = "Service Logs"
  filter = ["resource.type=\"cloud_run_revision\""]
}

module "http" {
  source = "../sections/http"
  title  = "HTTP"
  filter = ["resource.type=\"cloud_run_revision\""]
}

module "resources" {
  source = "../sections/resources"
  title  = "Resources"
  filter = ["resource.type=\"cloud_run_revision\""]
}

module "width" { source = "../sections/width" }

module "layout" {
  source   = "../sections/layout"
  sections = concat([
    for key in sort(keys(var.triggers)) : module.subscription[key].section
  ],
  [
    module.logs.section,
    module.http.section,
    module.resources.section,
  ])
}

resource "google_monitoring_dashboard" "dashboard" {
  dashboard_json = jsonencode({
    displayName = "Cloud Event Receiver: ${var.service_name}"
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
