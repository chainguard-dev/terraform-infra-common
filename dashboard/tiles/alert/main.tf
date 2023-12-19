variable "title" { type = string }
variable "alert_name" { type = string }

// https://cloud.google.com/monitoring/api/ref_v3/rest/v1/projects.dashboards#AlertChart
output "tile" {
  value = {
    title      = var.title
    alertChart = { name = var.alert_name }
  }
}
