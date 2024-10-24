module "width" { source = "../dashboard/sections/width" }

module "receiver-logs" {
  source = "../dashboard/sections/logs"
  title  = "Receiver Logs"
  filter = ["resource.type=\"cloud_run_revision\"", "resource.labels.service_name=\"${var.name}-rcv\""]
}

module "dispatcher-logs" {
  source = "../dashboard/sections/logs"
  title  = "Dispatcher Logs"
  filter = ["resource.type=\"cloud_run_revision\"", "resource.labels.service_name=\"${var.name}-dsp\""]
}

module "work-in-progress" {
  source = "../dashboard/widgets/xy"
  title  = "Amount of work in progress"
  filter = [
    "resource.type=\"prometheus_target\"",
    "metric.type=\"prometheus.googleapis.com/workqueue_in_progress_keys/gauge\"",
    "metric.label.\"service_name\"=\"${var.name}-dsp\"",
  ]
  group_by_fields = ["metric.label.\"service_name\""]
  primary_align   = "ALIGN_MAX"
  primary_reduce  = "REDUCE_MAX"

  thresholds = [var.concurrent-work]
}

module "work-queued" {
  source = "../dashboard/widgets/xy"
  title  = "Amount of work queued"
  filter = [
    "resource.type=\"prometheus_target\"",
    "metric.type=\"prometheus.googleapis.com/workqueue_queued_keys/gauge\"",
    "metric.label.\"service_name\"=\"${var.name}-dsp\"",
  ]
  group_by_fields = ["metric.label.\"service_name\""]
  primary_align   = "ALIGN_MEAN"
  primary_reduce  = "REDUCE_MEAN"
}

module "work-added" {
  source = "../dashboard/widgets/xy"
  title  = "Amount of work added"
  filter = [
    "resource.type=\"prometheus_target\"",
    "metric.type=\"prometheus.googleapis.com/workqueue_added_keys_total/counter\"",
    "metric.label.\"service_name\"=\"${var.name}-rcv\"",
  ]
  group_by_fields  = ["metric.label.\"service_name\""]
  primary_align    = "ALIGN_RATE"
  primary_reduce   = "REDUCE_NONE"
  secondary_align  = "ALIGN_NONE"
  secondary_reduce = "REDUCE_SUM"
}

module "process-latency" {
  source = "../dashboard/widgets/latency"
  title  = "Work processing latency"
  filter = [
    "resource.type=\"prometheus_target\"",
    "metric.type=\"prometheus.googleapis.com/workqueue_process_latency_seconds/histogram\"",
    "metric.label.\"service_name\"=\"${var.name}-dsp\"",
  ]
}

module "wait-latency" {
  source = "../dashboard/widgets/latency"
  title  = "Work wait times"
  filter = [
    "resource.type=\"prometheus_target\"",
    "metric.type=\"prometheus.googleapis.com/workqueue_wait_latency_seconds/histogram\"",
    "metric.label.\"service_name\"=\"${var.name}-dsp\"",
  ]
}

module "percent-deduped" {
  source    = "../dashboard/widgets/xy-ratio"
  title     = "Percentage of work deduplicated"
  legend    = ""
  plot_type = "LINE"

  numerator_filter = [
    "resource.type=\"prometheus_target\"",
    "metric.type=\"prometheus.googleapis.com/workqueue_deduped_keys_total/counter\"",
    "metric.label.\"service_name\"=\"${var.name}-rcv\"",
  ]
  denominator_filter = [
    "resource.type=\"prometheus_target\"",
    "metric.type=\"prometheus.googleapis.com/workqueue_added_keys_total/counter\"",
    "metric.label.\"service_name\"=\"${var.name}-rcv\"",
  ]

  alignment_period            = "60s"
  thresholds                  = []
  numerator_align             = "ALIGN_RATE"
  numerator_group_by_fields   = ["metric.label.\"service_name\""]
  numerator_reduce            = "REDUCE_SUM"
  denominator_align           = "ALIGN_RATE"
  denominator_group_by_fields = ["metric.label.\"service_name\""]
  denominator_reduce          = "REDUCE_SUM"
}

locals {
  columns = 3
  unit    = module.width.size / local.columns

  // https://www.terraform.io/language/functions/range
  // N columns, unit width each  ([0, unit, 2 * unit, ...])
  col = range(0, local.columns * local.unit, local.unit)

  tiles = [
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
  ]
}

module "collapsible" {
  source = "../dashboard/sections/collapsible"

  title     = "Workqueue State"
  tiles     = local.tiles
  collapsed = false
}

module "layout" {
  source = "../dashboard/sections/layout"

  sections = [
    module.collapsible.section,
    module.receiver-logs.section,
    module.dispatcher-logs.section,
  ]
}

module "dashboard" {
  source = "../dashboard"

  object = {
    displayName = "Cloud Workqueue: ${var.name}"
    labels = {
      "service" : ""
      "workqueue" : ""
    }

    // https://cloud.google.com/monitoring/api/ref_v3/rest/v1/projects.dashboards#mosaiclayout
    mosaicLayout = {
      columns = module.width.size
      tiles   = module.layout.tiles,
    }
  }
}

moved {
  from = google_monitoring_dashboard.dashboard
  to = module.dashboard.google_monitoring_dashboard.dashboard
}
