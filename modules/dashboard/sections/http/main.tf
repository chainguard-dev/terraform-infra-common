variable "title" { type = string }
variable "filter" { type = list(string) }
variable "collapsed" { default = false }
variable "service_name" { type = string }

module "width" { source = "../width" }

module "request_count" {
  source           = "../../widgets/xy"
  title            = "Request count"
  filter           = concat(var.filter, ["metric.type=\"run.googleapis.com/request_count\""])
  group_by_fields  = ["metric.label.\"response_code_class\""]
  primary_align    = "ALIGN_RATE"
  primary_reduce   = "REDUCE_NONE"
  secondary_align  = "ALIGN_NONE"
  secondary_reduce = "REDUCE_SUM"
}

module "incoming_latency" {
  source = "../../widgets/latency"
  title  = "Incoming request latency"
  filter = concat(var.filter, ["metric.type=\"run.googleapis.com/request_latencies\""])
}

locals {
  // gmp_filter is a subset of var.filter that does not include the "resource.type" string
  gmp_filter = [for f in var.filter : f if !strcontains(f, "resource.type")]
}

// TODO(mattmoor): output HTTP charts.
module "outbound_request_count" {
  source = "../../widgets/xy"
  title  = "Outbound Request count"
  filter = concat(local.gmp_filter, [
    "metric.type=\"prometheus.googleapis.com/http_client_request_count_total/counter\"",
    "metric.label.service_name=\"${var.service_name}\"",
  ])
  group_by_fields = [
    "metric.label.\"code\"",
    "metric.label.\"host\"",
  ]
  primary_align    = "ALIGN_RATE"
  primary_reduce   = "REDUCE_NONE"
  secondary_align  = "ALIGN_NONE"
  secondary_reduce = "REDUCE_SUM"
}

module "outbound_request_latency" {
  source = "../../widgets/latency"
  title  = "Outbound request latency"
  filter = concat(local.gmp_filter, [
    "metric.type=\"prometheus.googleapis.com/http_client_request_duration_seconds/histogram\"",
    "metric.label.service_name=\"${var.service_name}\"",
  ])
}

locals {
  columns = 2
  unit    = module.width.size / local.columns

  // https://www.terraform.io/language/functions/range
  // N columns, unit width each  ([0, unit, 2 * unit, ...])
  col = range(0, local.columns * local.unit, local.unit)

  tiles = [{
    yPos   = 0
    xPos   = local.col[0],
    height = local.unit,
    width  = local.unit,
    widget = module.request_count.widget,
    },
    {
      yPos   = 0
      xPos   = local.col[1],
      height = local.unit,
      width  = local.unit,
      widget = module.incoming_latency.widget,
    },
    {
      yPos   = local.unit
      xPos   = local.col[0],
      height = local.unit,
      width  = local.unit,
      widget = module.outbound_request_count.widget,
    },
    {
      yPos   = local.unit
      xPos   = local.col[1],
      height = local.unit,
      width  = local.unit,
      widget = module.outbound_request_latency.widget,
  }]
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
