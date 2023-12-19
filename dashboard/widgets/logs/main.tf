variable "title" { type = string }
variable "filter" { type = list(string) }

// https://cloud.google.com/monitoring/api/ref_v3/rest/v1/projects.dashboards#LogsPanel
output "tile" {
  value = {
    title     = var.title
    logsPanel = { filter = join("\n", var.filter) }
  }
}
