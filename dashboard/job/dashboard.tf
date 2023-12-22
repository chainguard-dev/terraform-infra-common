module "logs" {
  source = "../sections/logs"
  title  = "Job Logs"
  filter = ["resource.type=\"cloud_run_job\""]
}

module "resources" {
  source = "../sections/resources"
  title  = "Resources"
  filter = ["resource.type=\"cloud_run_job\""]
}

module "width" { source = "../sections/width" }

module "layout" {
  source = "../sections/layout"
  sections = [
    module.logs.section,
    module.resources.section,
  ]
}

resource "google_monitoring_dashboard" "dashboard" {
  dashboard_json = jsonencode({
    displayName = "Cloud Run Job: ${var.job_name}"
    labels = merge({
      "job" : ""
    }, var.labels)
    dashboardFilters = [{
      filterType  = "RESOURCE_LABEL"
      stringValue = var.job_name
      labelKey    = "job_name"
    }]

    // https://cloud.google.com/monitoring/api/ref_v3/rest/v1/projects.dashboards#mosaiclayout
    mosaicLayout = {
      columns = module.width.size
      tiles   = module.layout.tiles,
    }
  })
}
