variable "title" { type = string }
variable "filter" { type = list(string) }
variable "cloudrun_name" { type = string }
variable "collapsed" { default = false }
variable "notification_channels" {
  type = list(string)
}

module "width" { source = "../width" }

module "instance_count" {
  source          = "../../widgets/xy"
  title           = "Instance count + revisions"
  filter          = concat(var.filter, ["metric.type=\"run.googleapis.com/container/instance_count\""])
  group_by_fields = ["resource.label.\"revision_name\""]
  primary_align   = "ALIGN_MEAN"
  primary_reduce  = "REDUCE_SUM"
  plot_type       = "STACKED_AREA"
}

module "cpu_utilization" {
  source         = "../../widgets/xy"
  title          = "CPU utilization"
  filter         = concat(var.filter, ["metric.type=\"run.googleapis.com/container/cpu/utilizations\""])
  primary_align  = "ALIGN_DELTA"
  primary_reduce = "REDUCE_MEAN"
}

module "memory_utilization" {
  source         = "../../widgets/xy"
  title          = "Memory utilization"
  filter         = concat(var.filter, ["metric.type=\"run.googleapis.com/container/memory/utilizations\""])
  primary_align  = "ALIGN_DELTA"
  primary_reduce = "REDUCE_MEAN"
}

module "startup_latency" {
  source         = "../../widgets/xy"
  title          = "Startup latency"
  filter         = concat(var.filter, ["metric.type=\"run.googleapis.com/container/startup_latencies\""])
  primary_align  = "ALIGN_DELTA"
  primary_reduce = "REDUCE_MEAN"
  plot_type      = "STACKED_BAR"
}

module "sent_bytes" {
  source         = "../../widgets/xy"
  title          = "Sent bytes"
  filter         = concat(var.filter, ["metric.type=\"run.googleapis.com/container/network/sent_bytes_count\""])
  primary_align  = "ALIGN_MEAN"
  primary_reduce = "REDUCE_NONE"
}

module "received_bytes" {
  source         = "../../widgets/xy"
  title          = "Received bytes"
  filter         = concat(var.filter, ["metric.type=\"run.googleapis.com/container/network/received_bytes_count\""])
  primary_align  = "ALIGN_MEAN"
  primary_reduce = "REDUCE_NONE"
}

module "oom_alert" {
  source     = "../../widgets/alert"
  title      = google_monitoring_alert_policy.oom.display_name
  alert_name = google_monitoring_alert_policy.oom.name
}

resource "google_monitoring_alert_policy" "oom" {
  # In the absence of data, incident will auto-close after an hour
  alert_strategy {
    auto_close = "3600s"

    notification_rate_limit {
      period = "3600s" // re-alert once an hour if condition still valid.
    }
  }

  display_name = "${var.cloudrun_name} OOM Alert"
  combiner     = "OR"

  conditions {
    display_name = "${var.cloudrun_name} OOM Alert"

    condition_matched_log {
      filter = "${join(" AND ", var.filter)} AND logName:\"run.googleapis.com%2Fvarlog%2Fsystem\" AND severity=ERROR AND textPayload:\"Consider increasing the memory limit\""
    }
  }

  enabled = "true"

  notification_channels = var.notification_channels
}

locals {
  columns = 3
  unit    = module.width.size / local.columns

  // https://www.terraform.io/language/functions/range
  // N columns, unit width each  ([0, unit, 2 * unit, ...])
  col = range(0, local.columns * local.unit, local.unit)

  tiles = [{
    yPos   = 0,
    xPos   = local.col[0],
    height = local.unit,
    width  = module.width.size,
    widget = module.oom_alert.widget,
    },
    {
      yPos   = local.unit,
      xPos   = local.col[0],
      height = local.unit,
      width  = local.unit,
      widget = module.cpu_utilization.widget,
    },
    {
      yPos   = local.unit,
      xPos   = local.col[1],
      height = local.unit,
      width  = local.unit,
      widget = module.memory_utilization.widget,
    },
    {
      yPos   = local.unit,
      xPos   = local.col[2],
      height = local.unit,
      width  = local.unit,
      widget = module.instance_count.widget,
    },
    {
      yPos   = local.unit * 2,
      xPos   = local.col[0],
      height = local.unit,
      width  = local.unit,
      widget = module.startup_latency.widget,
    },
    {
      yPos   = local.unit * 2,
      xPos   = local.col[1],
      height = local.unit,
      width  = local.unit,
      widget = module.sent_bytes.widget,
    },
    {
      yPos   = local.unit * 2,
      xPos   = local.col[2],
      height = local.unit,
      width  = local.unit,
      widget = module.received_bytes.widget,
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
