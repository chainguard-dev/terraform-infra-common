locals {
  base_job_name = trimsuffix(var.job_name, "-cron")
  job_name      = "${local.base_job_name}-cron"
}

module "errgrp" {
  source       = "../sections/errgrp"
  title        = "Job Error Reporting"
  project_id   = var.project_id
  service_name = local.job_name
}

module "logs" {
  source = "../sections/logs"
  title  = "Job Logs"
  filter = ["resource.type=\"cloud_run_job\"", "resource.labels.job_name=\"${local.job_name}\""]
}

module "resources" {
  source                = "../sections/resources"
  title                 = "Resources"
  filter                = ["resource.type=\"cloud_run_job\"", "resource.labels.job_name=\"${local.job_name}\""]
  cloudrun_name         = local.job_name
  notification_channels = var.notification_channels
}

module "width" { source = "../sections/width" }

module "layout" {
  source = "../sections/layout"
  sections = [
    module.errgrp.section,
    module.logs.section,
    module.resources.section,
  ]
}

module "dashboard" {
  source = ".."

  object = {
    displayName = "Cloud Run Job: ${var.job_name}"
    labels = merge({
      "job" : ""
    }, var.labels)
    dashboardFilters = [{
      filterType  = "RESOURCE_LABEL"
      stringValue = local.job_name
      labelKey    = "job_name"
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
