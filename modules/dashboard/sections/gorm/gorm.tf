variable "title" { type = string }
variable "filter" { type = list(string) }
variable "collapsed" { default = false }
variable "service_name" { type = string }

module "width" { source = "../width" }

module "total_request_count" {
  source = "../../widgets/xy"
  title  = "GORM total request count"
  filter = concat(var.filter, [
    "resource.label.\"job\"=\"${var.service_name}\"",
    "resource.type=\"prometheus_target\"",
    "metric.type=\"prometheus.googleapis.com/gorm_calls_total/counter\"",
  ])
  group_by_fields = [
    "metric.label.\"code\""
  ]
  primary_align    = "ALIGN_RATE"
  primary_reduce   = "REDUCE_NONE"
  secondary_align  = "ALIGN_NONE"
  secondary_reduce = "REDUCE_SUM"
}

module "request_errors" {
  source = "../../widgets/xy"
  title  = "GORM error request count"
  filter = concat(var.filter, [
    "resource.label.\"job\"=\"${var.service_name}\"",
    "resource.type=\"prometheus_target\"",
    "metric.type=\"prometheus.googleapis.com/gorm_calls_total/counter\"",
    "metric.label.\"code\"=monitoring.regex.full_match(\"Error.*\")",
  ])
  group_by_fields = [
    "metric.label.\"table\"",
    "metric.label.\"code\"",
  ]
  primary_align    = "ALIGN_RATE"
  primary_reduce   = "REDUCE_NONE"
  secondary_align  = "ALIGN_NONE"
  secondary_reduce = "REDUCE_SUM"
}

module "table_request_count" {
  source = "../../widgets/xy"
  title  = "GORM table request count"
  filter = concat(var.filter, [
    "resource.label.\"job\"=\"${var.service_name}\"",
    "resource.type=\"prometheus_target\"",
    "metric.type=\"prometheus.googleapis.com/gorm_calls_total/counter\"",
  ])
  group_by_fields = [
    "metric.label.\"table\"",
    "metric.label.\"code\"",
  ]
  primary_align    = "ALIGN_RATE"
  primary_reduce   = "REDUCE_NONE"
  secondary_align  = "ALIGN_NONE"
  secondary_reduce = "REDUCE_SUM"
}

module "error_rate" {
  source = "../../widgets/percent"
  title  = "GORM Request error rate"
  legend = "Non-OK / All responses"

  common_filter = concat(var.filter, [
    "resource.label.\"job\"=\"${var.service_name}\"",
    "resource.type=\"prometheus_target\"",
    "metric.type=\"prometheus.googleapis.com/gorm_calls_total/counter\"",
  ])
  numerator_additional_filter = ["metric.label.\"code\"=monitoring.regex.full_match(\"Error.*\")"]
}

module "op_request_count" {
  source = "../../widgets/xy"
  title  = "GORM op request count"
  filter = concat(var.filter, [
    "resource.label.\"job\"=\"${var.service_name}\"",
    "resource.type=\"prometheus_target\"",
    "metric.type=\"prometheus.googleapis.com/gorm_calls_total/counter\"",
  ])
  group_by_fields = [
    "metric.label.\"op\"",
    "metric.label.\"code\"",
  ]
  primary_align    = "ALIGN_RATE"
  primary_reduce   = "REDUCE_NONE"
  secondary_align  = "ALIGN_NONE"
  secondary_reduce = "REDUCE_SUM"
}

module "open_connections" {
  source = "../../widgets/xy"
  title  = "GORM DB open connections"
  filter = concat(var.filter, [
    "resource.label.\"job\"=\"${var.service_name}\"",
    "resource.type=\"prometheus_target\"",
    "metric.type=\"prometheus.googleapis.com/gorm_dbstats_open_connections/gauge\"",
  ])
  group_by_fields  = []
  primary_align    = "ALIGN_MAX"
  primary_reduce   = "REDUCE_SUM"
  secondary_align  = "ALIGN_NONE"
  secondary_reduce = "REDUCE_SUM"
}

locals {
  columns = 2
  unit    = module.width.size / local.columns

  // https://www.terraform.io/language/functions/range
  // N columns, unit width each  ([0, unit, 2 * unit, ...])
  col = range(0, local.columns * local.unit, local.unit)

  tiles = [
    {
      yPos   = 0
      xPos   = local.col[0],
      height = local.unit,
      width  = local.unit,
      widget = module.total_request_count.widget,
      }, {
      yPos   = 0
      xPos   = local.col[1],
      height = local.unit,
      width  = local.unit,
      widget = module.request_errors.widget,
      }, {
      yPos   = local.unit
      xPos   = local.col[0],
      height = local.unit,
      width  = local.unit,
      widget = module.table_request_count.widget,
      }, {
      yPos   = local.unit
      xPos   = local.col[1],
      height = local.unit,
      width  = local.unit,
      widget = module.error_rate.widget,
      }, {
      yPos   = local.unit * 2
      xPos   = local.col[0],
      height = local.unit,
      width  = local.unit,
      widget = module.op_request_count.widget,
      }, {
      yPos   = local.unit * 2
      xPos   = local.col[1],
      height = local.unit,
      width  = local.unit,
      widget = module.open_connections.widget,
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
