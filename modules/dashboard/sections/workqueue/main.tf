variable "title" {
  type    = string
  default = "Workqueue State"
}
variable "filter" {
  type    = list(string)
  default = []
}
variable "collapsed" {
  type    = bool
  default = false
}
variable "service_name" {
  type = string
}
variable "receiver_name" {
  type    = string
  default = ""
}
variable "dispatcher_name" {
  type    = string
  default = ""
}
variable "max_retry" {
  type    = number
  default = 0
}
variable "concurrent_work" {
  type = number
}

locals {
  // Use provided names or derive from service_name
  rcv_name = var.receiver_name != "" ? var.receiver_name : "${var.service_name}-rcv"
  dsp_name = var.dispatcher_name != "" ? var.dispatcher_name : "${var.service_name}-dsp"

  // gmp_filter is a subset of var.filter that does not include the "resource.type" string
  gmp_filter = [for f in var.filter : f if !strcontains(f, "resource.type")]
}

module "width" { source = "../width" }

module "work-in-progress" {
  source = "../../widgets/xy"
  title  = "Amount of work in progress"
  filter = concat(local.gmp_filter, [
    "resource.type=\"prometheus_target\"",
    "metric.type=\"prometheus.googleapis.com/workqueue_in_progress_keys/gauge\"",
    "metric.label.\"service_name\"=\"${local.dsp_name}\"",
  ])
  group_by_fields = null
  primary_align   = "ALIGN_MAX"
  primary_reduce  = "REDUCE_MAX"
  thresholds      = [var.concurrent_work]
}

module "work-queued" {
  source = "../../widgets/xy"
  title  = "Amount of work queued"
  filter = concat(local.gmp_filter, [
    "resource.type=\"prometheus_target\"",
    "metric.type=\"prometheus.googleapis.com/workqueue_queued_keys/gauge\"",
    "metric.label.\"service_name\"=\"${local.dsp_name}\"",
  ])
  group_by_fields = null
  plot_type       = "STACKED_AREA"
  primary_align   = "ALIGN_MIN"
  primary_reduce  = "REDUCE_MIN"
}

module "work-added" {
  source = "../../widgets/xy"
  title  = "Amount of work added"
  filter = concat(local.gmp_filter, [
    "resource.type=\"prometheus_target\"",
    "metric.type=\"prometheus.googleapis.com/workqueue_added_keys_total/counter\"",
    "metric.label.\"service_name\"=\"${local.rcv_name}\"",
  ])
  group_by_fields = ["resource.label.\"location\""]
  plot_type       = "STACKED_AREA"
  primary_align   = "ALIGN_RATE"
  primary_reduce  = "REDUCE_SUM"
}

module "process-latency" {
  source = "../../widgets/latency"
  title  = "Work processing latency"
  filter = concat(local.gmp_filter, [
    "resource.type=\"prometheus_target\"",
    "metric.type=\"prometheus.googleapis.com/workqueue_process_latency_seconds/histogram\"",
    "metric.label.\"service_name\"=\"${local.dsp_name}\"",
  ])
  group_by_fields = ["resource.label.\"location\""]
}

module "wait-latency" {
  source = "../../widgets/latency"
  title  = "Work wait times"
  filter = concat(local.gmp_filter, [
    "resource.type=\"prometheus_target\"",
    "metric.type=\"prometheus.googleapis.com/workqueue_wait_latency_from_scheduled_seconds/histogram\"",
    "metric.label.\"service_name\"=\"${local.dsp_name}\"",
  ])
  group_by_fields = null
}

module "percent-deduped" {
  source    = "../../widgets/xy-ratio"
  title     = "Percentage of work deduplicated"
  legend    = ""
  plot_type = "LINE"

  numerator_filter = concat(local.gmp_filter, [
    "resource.type=\"prometheus_target\"",
    "metric.type=\"prometheus.googleapis.com/workqueue_deduped_keys_total/counter\"",
    "metric.label.\"service_name\"=\"${local.rcv_name}\"",
  ])
  denominator_filter = concat(local.gmp_filter, [
    "resource.type=\"prometheus_target\"",
    "metric.type=\"prometheus.googleapis.com/workqueue_added_keys_total/counter\"",
    "metric.label.\"service_name\"=\"${local.rcv_name}\"",
  ])

  alignment_period            = "60s"
  thresholds                  = []
  numerator_align             = "ALIGN_RATE"
  numerator_group_by_fields   = null
  numerator_reduce            = "REDUCE_SUM"
  denominator_align           = "ALIGN_RATE"
  denominator_group_by_fields = null
  denominator_reduce          = "REDUCE_SUM"
}

module "attempts-at-completion" {
  source = "../../widgets/xy"
  title  = "Attempts at completion (p95)"
  filter = concat(local.gmp_filter, [
    "resource.type=\"prometheus_target\"",
    "metric.type=\"prometheus.googleapis.com/workqueue_attempts_at_completion/histogram\"",
    "metric.label.\"service_name\"=\"${local.dsp_name}\"",
  ])
  primary_align  = "ALIGN_DELTA"
  primary_reduce = "REDUCE_PERCENTILE_95"
  thresholds     = var.max_retry > 0 ? [var.max_retry] : []
}

module "max-attempts" {
  source = "../../widgets/xy"
  title  = "Maximum task attempts"
  filter = concat(local.gmp_filter, [
    "resource.type=\"prometheus_target\"",
    "metric.type=\"prometheus.googleapis.com/workqueue_max_attempts/gauge\"",
    "metric.label.\"service_name\"=\"${local.dsp_name}\"",
  ])
  group_by_fields = null
  primary_align   = "ALIGN_MAX"
  primary_reduce  = "REDUCE_MAX"
  thresholds      = var.max_retry > 0 ? [var.max_retry] : []
}

module "time-to-completion" {
  source = "../../widgets/xy"
  title  = "Time to completion (95p by priority)"
  filter = concat(local.gmp_filter, [
    "resource.type=\"prometheus_target\"",
    "metric.type=\"prometheus.googleapis.com/workqueue_time_to_completion_seconds/histogram\"",
    "metric.label.\"service_name\"=\"${local.dsp_name}\"",
  ])
  group_by_fields = ["metric.label.\"priority_class\""]
  primary_align   = "ALIGN_DELTA"
  primary_reduce  = "REDUCE_PERCENTILE_95"
}

module "dead-letter-queue" {
  count  = var.max_retry > 0 ? 1 : 0
  source = "../../widgets/xy"
  title  = "Dead-letter queue size"
  filter = concat(local.gmp_filter, [
    "resource.type=\"prometheus_target\"",
    "metric.type=\"prometheus.googleapis.com/workqueue_dead_lettered_keys/gauge\"",
    "metric.label.\"service_name\"=\"${local.dsp_name}\"",
  ])
  group_by_fields = null
  plot_type       = "STACKED_AREA"
  primary_align   = "ALIGN_MAX"
  primary_reduce  = "REDUCE_MAX"
}

module "lease-age" {
  source = "../../widgets/latency"
  title  = "Active lease age (p95)"
  filter = concat(local.gmp_filter, [
    "resource.type=\"prometheus_target\"",
    "metric.type=\"prometheus.googleapis.com/workqueue_lease_age_seconds/histogram\"",
    "metric.label.\"service_name\"=\"${local.dsp_name}\"",
  ])
  group_by_fields = null
}

module "expired-leases" {
  source = "../../widgets/xy"
  title  = "Expired leases rate"
  filter = concat(local.gmp_filter, [
    "resource.type=\"prometheus_target\"",
    "metric.type=\"prometheus.googleapis.com/workqueue_expired_leases_total/counter\"",
    "metric.label.\"service_name\"=\"${local.dsp_name}\"",
  ])
  group_by_fields = ["resource.label.\"location\""]
  plot_type       = "STACKED_AREA"
  primary_align   = "ALIGN_RATE"
  primary_reduce  = "REDUCE_SUM"
}

module "time-until-eligible" {
  source = "../../widgets/latency"
  title  = "Time until eligible (p95)"
  filter = concat(local.gmp_filter, [
    "resource.type=\"prometheus_target\"",
    "metric.type=\"prometheus.googleapis.com/workqueue_time_until_eligible_seconds/histogram\"",
    "metric.label.\"service_name\"=\"${local.dsp_name}\"",
  ])
  group_by_fields = null
}

module "enumerate-latency" {
  source = "../../widgets/latency"
  title  = "Enumerate latency (p95)"
  filter = concat(local.gmp_filter, [
    "resource.type=\"prometheus_target\"",
    "metric.type=\"prometheus.googleapis.com/workqueue_enumerate_latency_seconds/histogram\"",
    "metric.label.\"service_name\"=\"${local.dsp_name}\"",
  ])
  group_by_fields = ["resource.label.\"location\""]
}

locals {
  columns = 3
  unit    = module.width.size / local.columns

  // https://www.terraform.io/language/functions/range
  // N columns, unit width each  ([0, unit, 2 * unit, ...])
  col = range(0, local.columns * local.unit, local.unit)

  tiles = concat([
    {
      yPos   = 0,
      xPos   = local.col[0],
      height = local.unit,
      width  = local.unit,
      widget = module.work-in-progress.widget,
    },
    {
      yPos   = 0,
      xPos   = local.col[1],
      height = local.unit,
      width  = local.unit,
      widget = module.work-queued.widget,
    },
    {
      yPos   = 0,
      xPos   = local.col[2],
      height = local.unit,
      width  = local.unit,
      widget = module.work-added.widget,
    },
    {
      yPos   = local.unit,
      xPos   = local.col[0],
      height = local.unit,
      width  = local.unit,
      widget = module.process-latency.widget,
    },
    {
      yPos   = local.unit,
      xPos   = local.col[1],
      height = local.unit,
      width  = local.unit,
      widget = module.wait-latency.widget,
    },
    {
      yPos   = local.unit,
      xPos   = local.col[2],
      height = local.unit,
      width  = local.unit,
      widget = module.percent-deduped.widget,
    },
    {
      yPos   = local.unit * 2,
      xPos   = local.col[0],
      height = local.unit,
      width  = local.unit,
      widget = module.attempts-at-completion.widget,
    },
    {
      yPos   = local.unit * 2,
      xPos   = local.col[1],
      height = local.unit,
      width  = local.unit,
      widget = module.max-attempts.widget,
    },
    {
      yPos   = local.unit * 3,
      xPos   = local.col[0],
      height = local.unit,
      width  = local.unit,
      widget = module.time-to-completion.widget,
    },
    {
      yPos   = local.unit * 3,
      xPos   = local.col[1],
      height = local.unit,
      width  = local.unit,
      widget = module.lease-age.widget,
    },
    {
      yPos   = local.unit * 3,
      xPos   = local.col[2],
      height = local.unit,
      width  = local.unit,
      widget = module.expired-leases.widget,
    },
    {
      yPos   = local.unit * 4,
      xPos   = local.col[0],
      height = local.unit,
      width  = local.unit,
      widget = module.enumerate-latency.widget,
    },
    {
      yPos   = local.unit * 4,
      xPos   = local.col[1],
      height = local.unit,
      width  = local.unit,
      widget = module.time-until-eligible.widget,
    }
    ],
    var.max_retry > 0 ? [
      {
        yPos   = local.unit * 2,
        xPos   = local.col[2],
        height = local.unit,
        width  = local.unit,
        widget = module.dead-letter-queue[0].widget,
      }
  ] : [])
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
