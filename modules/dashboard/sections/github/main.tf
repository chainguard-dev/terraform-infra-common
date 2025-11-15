variable "title" { type = string }
variable "filter" { type = list(string) }
variable "collapsed" { default = false }

module "width" { source = "../width" }

module "used" {
  source = "../../widgets/xy"
  title  = "GitHub API Rate Limit Used (%)"
  filter = concat(var.filter, [
    "metric.type=\"prometheus.googleapis.com/github_rate_limit_used/gauge\"",
  ])
  group_by_fields = ["metric.label.\"resource\"", "metric.label.\"organization\""]
  primary_align   = "ALIGN_MAX"
  primary_reduce  = "REDUCE_MAX"
}

module "time_to_reset" {
  source = "../../widgets/xy"
  title  = "Time to next GitHub API reset"
  filter = concat(var.filter, [
    "metric.type=\"prometheus.googleapis.com/github_rate_limit_time_to_reset/gauge\"",
  ])
  group_by_fields = ["metric.label.\"resource\"", "metric.label.\"organization\""]
  primary_align   = "ALIGN_MIN"
  primary_reduce  = "REDUCE_MIN"
}

module "api_calls" {
  source = "../../widgets/xy"
  title  = "GitHub API Calls by Endpoint"
  filter = concat(var.filter, [
    "metric.type=\"prometheus.googleapis.com/http_client_request_count_total/counter\"",
    "resource.type=\"prometheus_target\"",
    "metric.label.\"host\"=\"api.github.com\"",
  ])
  group_by_fields = [
    "metric.label.\"path\"",
    "metric.label.\"method\"",
  ]
  plot_type      = "STACKED_BAR"
  primary_align  = "ALIGN_RATE"
  primary_reduce = "REDUCE_SUM"
}

locals {
  columns = 3
  unit    = module.width.size / local.columns

  // https://www.terraform.io/language/functions/range
  // N columns, unit width each  ([0, unit, 2 * unit, ...])
  col = range(0, local.columns * local.unit, local.unit)

  tiles = [
    {
      yPos   = 0,
      xPos   = local.col[0],
      height = local.unit,
      width  = local.unit,
      widget = module.api_calls.widget,
    },
    {
      yPos   = 0,
      xPos   = local.col[1],
      height = local.unit,
      width  = local.unit,
      widget = module.used.widget,
    },
    {
      yPos   = 0,
      xPos   = local.col[2],
      height = local.unit,
      width  = local.unit,
      widget = module.time_to_reset.widget,
    },
  ]
}

module "collapsible" {
  source = "../collapsible"

  title     = var.title
  tiles     = local.tiles
  collapsed = var.collapsed
}

output "section" {
  value = module.collapsible.section
}
