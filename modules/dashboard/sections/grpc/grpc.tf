variable "title" { type = string }
variable "filter" { type = list(string) }
variable "collapsed" { default = false }
variable "service_name" { type = string }
variable "grpc_non_error_codes" {
  description = "List of grpc codes to not counted as error, case-sensitive."
  type        = list(string)
  default = [
    "OK",
    "Aborted",
    "AlreadyExists",
    "Canceled",
    "NotFound",
  ]
}

module "width" { source = "../width" }

module "request_count" {
  source = "../../widgets/xy"
  title  = "Request count"
  filter = concat(var.filter, [
    "metric.type=\"prometheus.googleapis.com/grpc_server_handled_total/counter\"",
    "resource.type=\"prometheus_target\"",
  ])
  group_by_fields = [
    "metric.label.\"grpc_service\"",
    "metric.label.\"grpc_method\"",
    "metric.label.\"grpc_code\""
  ]
  primary_align  = "ALIGN_RATE"
  primary_reduce = "REDUCE_SUM"
}

module "failure_rate" {
  source = "../../widgets/percent"
  title  = "Request failure rate"
  legend = "Non-[Aborted|AlreadyExists|Canceled|NotFound] resposnes / All responses"

  common_filter = concat(var.filter, [
    "metric.type=\"prometheus.googleapis.com/grpc_server_handled_total/counter\"",
    "resource.type=\"prometheus_target\"",
  ])
  numerator_additional_filter = [
    "metric.label.\"grpc_code\"!=monitoring.regex.full_match(\"${join("|", var.grpc_non_error_codes)}\")"
  ]
}

module "incoming_latency" {
  source = "../../widgets/latency"
  title  = "Incoming request latency"
  filter = concat(var.filter, [
    "metric.type=\"prometheus.googleapis.com/grpc_server_handling_seconds/histogram\"",
    "resource.type=\"prometheus_target\"",
  ])
  group_by_fields = [
    "metric.label.\"grpc_service\"",
    "metric.label.\"grpc_method\"",
  ]
}

module "outbound_request_count" {
  source = "../../widgets/xy"
  title  = "Outbound request count"
  filter = concat(var.filter, [
    "metric.type=\"prometheus.googleapis.com/grpc_client_handled_total/counter\"",
    "resource.type=\"prometheus_target\"",
  ])
  group_by_fields = [
    "metric.label.\"grpc_service\"",
    "metric.label.\"grpc_method\"",
    "metric.label.\"grpc_code\""
  ]
  primary_align  = "ALIGN_RATE"
  primary_reduce = "REDUCE_SUM"
}

module "outbound_latency" {
  source = "../../widgets/latency"
  title  = "Outbound request latency"
  filter = concat(var.filter, [
    "metric.type=\"prometheus.googleapis.com/grpc_client_handling_seconds/histogram\"",
    "resource.type=\"prometheus_target\"",
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
      }, {
      yPos   = local.unit
      xPos   = local.col[0],
      height = local.unit,
      width  = local.unit,
      widget = module.outbound_request_count.widget,
      }, {
      yPos   = local.unit
      xPos   = local.col[1],
      height = local.unit,
      width  = local.unit,
      widget = module.outbound_latency.widget,
      }, {
      yPos   = local.unit * 2
      xPos   = local.col[0],
      height = local.unit,
      width  = local.unit,
      widget = module.failure_rate.widget,
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
