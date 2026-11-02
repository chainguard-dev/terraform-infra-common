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
variable "cloudrun_type" {
  description = "A job serves no requests, so \"job\" charts outbound calls only."
  type        = string
  default     = "service"

  validation {
    condition     = contains(["service", "job"], var.cloudrun_type)
    error_message = "Allowed values for 'cloudrun_type' are 'service' or 'job'."
  }
}

module "width" { source = "../width" }

locals {
  // Incoming requests are Cloud Run metrics, scoped by the service_name
  // resource label. Outbound calls are Prometheus metrics, scoped by the
  // service_name label the otel sidecar stamps, and gmp_filter drops
  // var.filter's "resource.type" strings for them.
  run_filter = concat(var.filter, ["resource.label.\"service_name\"=\"${var.service_name}\""])
  gmp_filter = concat(
    [for f in var.filter : f if !strcontains(f, "resource.type")],
    ["metric.label.\"service_name\"=\"${var.service_name}\""],
  )
}

module "request_count" {
  source          = "../../widgets/xy"
  title           = "Incoming request count"
  filter          = concat(local.run_filter, ["resource.type=\"cloud_run_revision\"", "metric.type=\"run.googleapis.com/request_count\""])
  group_by_fields = ["metric.label.\"response_code_class\""]
  primary_align   = "ALIGN_RATE"
  primary_reduce  = "REDUCE_SUM"
}

module "incoming_latency" {
  source = "../../widgets/latency"
  title  = "Incoming request latency"
  filter = concat(local.run_filter, ["resource.type=\"cloud_run_revision\"", "metric.type=\"run.googleapis.com/request_latencies\""])
}

// Each failing code is charted as its share of all requests, stacked so the
// top of the stack is the overall failure rate.
module "incoming_failure_rate" {
  source    = "../../widgets/percent"
  title     = "Incoming request failure rate"
  legend    = "$${metric.labels.response_code}"
  plot_type = "STACKED_AREA"

  common_filter = concat(local.run_filter, [
    "metric.type=\"run.googleapis.com/request_count\"",
    "resource.type=\"cloud_run_revision\"",
  ])
  numerator_additional_filter = ["metric.label.\"response_code_class\"=\"5xx\""]
  numerator_group_by_fields   = ["metric.label.\"response_code\""]
}

module "outbound_request_count" {
  source = "../../widgets/xy"
  title  = "Outbound request count"
  filter = concat(local.gmp_filter, [
    "metric.type=\"prometheus.googleapis.com/http_client_request_count_total/counter\"",
    "resource.type=\"prometheus_target\"",
  ])
  group_by_fields = [
    "metric.label.\"code\"",
    "metric.label.\"host\"",
  ]
  primary_align  = "ALIGN_RATE"
  primary_reduce = "REDUCE_SUM"
}

module "outbound_request_latency" {
  source = "../../widgets/latency"
  title  = "Outbound request latency"
  filter = concat(local.gmp_filter, [
    "metric.type=\"prometheus.googleapis.com/http_client_request_duration_seconds/histogram\"",
    "resource.type=\"prometheus_target\"",
  ])
  group_by_fields = [
    "metric.label.\"host\"",
  ]
}

// A request that never got a response records a transport error such as
// "io-timeout" as its code, so failures are everything outside 1xx-4xx.
module "outbound_failure_rate" {
  source    = "../../widgets/percent"
  title     = "Outbound request failure rate"
  legend    = "$${metric.labels.code}"
  plot_type = "STACKED_AREA"

  common_filter = concat(local.gmp_filter, [
    "metric.type=\"prometheus.googleapis.com/http_client_request_count_total/counter\"",
    "resource.type=\"prometheus_target\"",
  ])
  numerator_additional_filter = ["metric.label.\"code\"!=monitoring.regex.full_match(\"[1-4][0-9][0-9]\")"]
  numerator_group_by_fields   = ["metric.label.\"code\""]
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

  incoming = [module.request_count.widget, module.incoming_latency.widget, module.incoming_failure_rate.widget]
  outbound = [module.outbound_request_count.widget, module.outbound_request_latency.widget, module.outbound_failure_rate.widget]

  // A job serves no requests, so it has no incoming row.
  rows = concat(var.cloudrun_type == "service" ? [local.incoming] : [], [local.outbound])

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
