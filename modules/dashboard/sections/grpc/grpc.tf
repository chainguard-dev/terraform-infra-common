terraform {
  required_version = ">= 1.9"
}

variable "title" { type = string }
variable "filter" { type = list(string) }
variable "collapsed" {
  type    = bool
  default = false
}
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

locals {
  // The otel sidecar stamps the Cloud Run service or job name as service_name.
  filter = concat(var.filter, ["metric.label.\"service_name\"=\"${var.service_name}\""])
}

module "request_count" {
  source = "../../widgets/xy"
  title  = "Incoming request count"
  filter = concat(local.filter, [
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

module "incoming_latency" {
  source = "../../widgets/latency"
  title  = "Incoming request latency"
  filter = concat(local.filter, [
    "metric.type=\"prometheus.googleapis.com/grpc_server_handling_seconds/histogram\"",
    "resource.type=\"prometheus_target\"",
  ])
  group_by_fields = [
    "metric.label.\"grpc_service\"",
    "metric.label.\"grpc_method\"",
  ]
}

// Each failing code is charted as its share of all requests, stacked so the
// top of the stack is the overall failure rate.
module "incoming_failure_rate" {
  source    = "../../widgets/percent"
  title     = "Incoming request failure rate"
  legend    = "$${metric.labels.grpc_code}"
  plot_type = "STACKED_AREA"

  common_filter = concat(local.filter, [
    "metric.type=\"prometheus.googleapis.com/grpc_server_handled_total/counter\"",
    "resource.type=\"prometheus_target\"",
  ])
  numerator_additional_filter = [
    "metric.label.\"grpc_code\"!=monitoring.regex.full_match(\"${join("|", var.grpc_non_error_codes)}\")"
  ]
  numerator_group_by_fields = ["metric.label.\"grpc_code\""]
}

module "outbound_request_count" {
  source = "../../widgets/xy"
  title  = "Outbound request count"
  filter = concat(local.filter, [
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
  filter = concat(local.filter, [
    "metric.type=\"prometheus.googleapis.com/grpc_client_handling_seconds/histogram\"",
    "resource.type=\"prometheus_target\"",
  ])
  group_by_fields = [
    "metric.label.\"grpc_service\"",
    "metric.label.\"grpc_method\"",
  ]
}

module "outbound_failure_rate" {
  source    = "../../widgets/percent"
  title     = "Outbound request failure rate"
  legend    = "$${metric.labels.grpc_code}"
  plot_type = "STACKED_AREA"

  common_filter = concat(local.filter, [
    "metric.type=\"prometheus.googleapis.com/grpc_client_handled_total/counter\"",
    "resource.type=\"prometheus_target\"",
  ])
  numerator_additional_filter = [
    "metric.label.\"grpc_code\"!=monitoring.regex.full_match(\"${join("|", var.grpc_non_error_codes)}\")"
  ]
  numerator_group_by_fields = ["metric.label.\"grpc_code\""]
}

locals {
  // One row per direction: count, latency, failure rate.
  columns = 3
  unit    = module.width.size / local.columns
  // Keep the height of the old two-column grid; a square tile at a third of
  // the width is too short to read charts with many series.
  height = module.width.size / 2

  // https://www.terraform.io/language/functions/range
  // N columns, unit width each  ([0, unit, 2 * unit, ...])
  col = range(0, local.columns * local.unit, local.unit)

  rows = [
    [module.request_count.widget, module.incoming_latency.widget, module.incoming_failure_rate.widget],
    [module.outbound_request_count.widget, module.outbound_latency.widget, module.outbound_failure_rate.widget],
  ]

  tiles = flatten([
    for y, row in local.rows : [
      for x, widget in row : {
        yPos   = y * local.height
        xPos   = local.col[x]
        height = local.height
        width  = local.unit
        widget = widget
      }
    ]
  ])
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
