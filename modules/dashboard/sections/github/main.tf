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

locals {
  columns = 3
  unit    = module.width.size / local.columns

  // https://www.terraform.io/language/functions/range
  // N columns, unit width each  ([0, unit, 2 * unit, ...])
  col = range(0, local.columns * local.unit, local.unit)

  tiles = [
    {
      yPos   = local.unit,
      xPos   = local.col[0],
      height = local.unit,
      width  = 2 * local.unit,
      widget = module.used.widget,
    },
    {
      yPos   = local.unit,
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
