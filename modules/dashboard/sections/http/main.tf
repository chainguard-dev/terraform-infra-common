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
  title           = "Request count"
  filter          = concat(local.run_filter, ["resource.type=\"cloud_run_revision\"", "metric.type=\"run.googleapis.com/request_count\""])
  group_by_fields = ["metric.label.\"response_code_class\""]
  primary_align   = "ALIGN_RATE"
  primary_reduce  = "REDUCE_SUM"
}

module "failure_rate" {
  source = "../../widgets/percent"
  title  = "Request failure rate"
  legend = "5xx responses / All responses"

  common_filter = concat(local.run_filter, [
    "metric.type=\"run.googleapis.com/request_count\"",
    "resource.type=\"cloud_run_revision\"",
  ])
  numerator_additional_filter = ["metric.label.\"response_code_class\"=\"5xx\""]
}

module "incoming_latency" {
  source = "../../widgets/latency"
  title  = "Incoming request latency"
  filter = concat(local.run_filter, ["resource.type=\"cloud_run_revision\"", "metric.type=\"run.googleapis.com/request_latencies\""])
}

// TODO(mattmoor): output HTTP charts.
module "outbound_request_count" {
  source = "../../widgets/xy"
  title  = "Outbound Request count"
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

locals {
  columns = 2
  unit    = module.width.size / local.columns

  // https://www.terraform.io/language/functions/range
  // N columns, unit width each  ([0, unit, 2 * unit, ...])
  col = range(0, local.columns * local.unit, local.unit)

  incoming   = var.cloudrun_type == "service"
  outbound_y = local.incoming ? local.unit : 0

  // The widgets differ in type, so each optional tile gets its own conditional.
  tiles = concat(
    local.incoming ? [{
      yPos   = 0
      xPos   = local.col[0],
      height = local.unit,
      width  = local.unit,
      widget = module.request_count.widget,
    }] : [],
    local.incoming ? [{
      yPos   = 0
      xPos   = local.col[1],
      height = local.unit,
      width  = local.unit,
      widget = module.incoming_latency.widget,
    }] : [],
    [
      {
        yPos   = local.outbound_y
        xPos   = local.col[0],
        height = local.unit,
        width  = local.unit,
        widget = module.outbound_request_count.widget,
      },
      {
        yPos   = local.outbound_y
        xPos   = local.col[1],
        height = local.unit,
        width  = local.unit,
        widget = module.outbound_request_latency.widget,
      },
    ],
    local.incoming ? [{
      yPos   = local.unit * 2
      xPos   = local.col[0],
      height = local.unit,
      width  = local.unit,
      widget = module.failure_rate.widget,
    }] : [],
  )
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
