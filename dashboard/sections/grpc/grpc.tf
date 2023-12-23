variable "title" { type = string }
variable "filter" { type = list(string) }
variable "collapsed" { default = false }
variable "grpc_service_name" { type = string }

module "width" { source = "../width" }

module "request_count" {
  source = "../../widgets/xy"
  title  = "Request count"
  filter = concat(var.filter, [
    "metric.type=\"prometheus.googleapis.com/grpc_server_handled_total/counter\"",
    "metric.label.grpc_service=monitoring.regex.full_match(\"${var.grpc_service_name}.*\")",
  ])
  group_by_fields = [
    "metric.label.\"grpc_service\"",
    "metric.label.\"grpc_method\"",
    "metric.label.\"grpc_code\""
  ]
  primary_align    = "ALIGN_RATE"
  primary_reduce   = "REDUCE_NONE"
  secondary_align  = "ALIGN_NONE"
  secondary_reduce = "REDUCE_SUM"
}

module "incoming_latency" {
  source = "../../widgets/latency"
  title  = "Incoming request latency"
  filter = concat(var.filter, [
    "metric.type=\"prometheus.googleapis.com/grpc_server_handling_seconds/histogram\"",
    "metric.label.\"grpc_service\"=monitoring.regex.full_match(\"${var.grpc_service_name}.*\")",
  ])
  group_by_fields = [
    "metric.label.\"grpc_service\"",
    "metric.label.\"grpc_method\"",
  ]
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
      widget = module.request_count.widget,
      }, {
      yPos   = 0
      xPos   = local.col[1],
      height = local.unit,
      width  = local.unit,
      widget = module.incoming_latency.widget,
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
