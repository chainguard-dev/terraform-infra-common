variable "title" { type = string }
variable "content" { type = string }

// https://cloud.google.com/monitoring/api/ref_v3/rest/v1/projects.dashboards#Text
output "widget" {
  value = {
    title   = var.title
    format  = "MARKDOWN"
    content = var.content
  }
}
