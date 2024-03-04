variable "title" { type = string }
variable "filter" { type = list(string) }
variable "cloudrun_name" { type = string }
variable "collapsed" { default = false }

module "width" { source = "../width" }

module "remaining" {
  source = "../../widgets/xy"
  title  = "GitHub API Rate Limit Remaining"
  filter = concat(var.filter, [
    "metric.type=\"prometheus.googleapis.com/github_rate_limit_remaining/gauge\"",
    "metric.label.service_name=\"${var.cloudrun_name}\"",
  ])
  group_by_fields = ["resource.label.\"resource\""]
  primary_align   = "ALIGN_MEAN"
  primary_reduce  = "REDUCE_SUM"
  plot_type       = "STACKED_AREA"
}

module "limit" {
  source = "../../widgets/xy"
  title  = "GitHub API Rate Limit"
  filter = concat(var.filter, [
    "metric.type=\"prometheus.googleapis.com/github_rate_limit/gauge\"",
    "metric.label.service_name=\"${var.cloudrun_name}\"",
  ])
  group_by_fields = ["resource.label.\"resource\""]
  primary_align   = "ALIGN_DELTA"
  primary_reduce  = "REDUCE_MEAN"
}

module "reset" {
  source = "../../widgets/xy"
  title  = "Next GitHub API reset"
  filter = concat(var.filter, [
    "metric.type=\"prometheus.googleapis.com/github_rate_limit_remaining_reset/gauge\"",
    "metric.label.service_name=\"${var.cloudrun_name}\"",
  ])
  group_by_fields = ["resource.label.\"resource\""]
  primary_align   = "ALIGN_DELTA"
  primary_reduce  = "REDUCE_MEAN"
}

locals {
  columns = 3
  unit    = module.width.size / local.columns

  // https://www.terraform.io/language/functions/range
  // N columns, unit width each  ([0, unit, 2 * unit, ...])
  col = range(0, local.columns * local.unit, local.unit)

  tiles = [{

    yPos   = local.unit,
    xPos   = local.col[0],
    height = local.unit,
    width  = local.unit,
    widget = module.remaining.widget,
    },
    {
      yPos   = local.unit,
      xPos   = local.col[1],
      height = local.unit,
      width  = local.unit,
      widget = module.limit.widget,
    },
    {
      yPos   = local.unit,
      xPos   = local.col[2],
      height = local.unit,
      width  = local.unit,
      widget = module.reset.widget,
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
