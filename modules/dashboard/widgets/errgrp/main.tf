variable "title" { type = string }
variable "project_id" { type = string }
variable "service_name" { type = string }

// https://cloud.google.com/monitoring/api/ref_v3/rest/v1/projects.dashboards#ErrorReportingPanel
output "widget" {
  value = {
    title = var.title
    errorReportingPanel = {
      projectNames = ["projects/${var.project_id}"]
      services     = [var.service_name]
    }
  }
}
