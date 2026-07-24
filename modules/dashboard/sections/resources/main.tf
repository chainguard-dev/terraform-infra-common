variable "title" { type = string }
variable "filter" { type = list(string) }
variable "cloudrun_name" { type = string }
variable "cloudrun_type" {
  type    = string
  default = "service"

  validation {
    condition     = contains(["service", "job"], var.cloudrun_type)
    error_message = "Allowed values for 'cloudrun_type' are 'service' or 'job'."
  }
}
variable "collapsed" { default = false }
variable "notification_channels" {
  type = list(string)
}

locals {
  filter = concat(var.filter, var.cloudrun_type == "job" ? ["resource.type=\"cloud_run_job\""] : ["resource.type=\"cloud_run_revision\""])
}

module "width" { source = "../width" }

module "instance_count" {
  source          = "../../widgets/xy"
  title           = "Instance count + revisions"
  filter          = concat(local.filter, ["metric.type=\"run.googleapis.com/container/instance_count\""])
  group_by_fields = ["resource.label.\"revision_name\""]
  primary_align   = "ALIGN_MEAN"
  primary_reduce  = "REDUCE_SUM"
  plot_type       = "STACKED_AREA"
}

module "cpu_utilization" {
  source         = "../../widgets/xy"
  title          = "CPU utilization"
  filter         = concat(local.filter, ["metric.type=\"run.googleapis.com/container/cpu/utilizations\""])
  primary_align  = "ALIGN_DELTA"
  primary_reduce = "REDUCE_PERCENTILE_99"
}

module "disk_usage" {
  source = "../../widgets/xy"
  title  = "Disk usage"
  filter = concat(var.filter, [
    "metric.type=\"prometheus.googleapis.com/disk_usage_bytes/gauge\"",
    "resource.type=\"prometheus_target\"",
  ])
  primary_align  = "ALIGN_MEAN"
  primary_reduce = "REDUCE_PERCENTILE_99"
}

// The run.googleapis.com/instance/disk/* metrics only report on mounted
// volumes (e.g. disk-backed emptyDir), broken down by volume name.
module "instance_disk_utilization" {
  source          = "../../widgets/xy"
  title           = "Instance disk utilization"
  filter          = concat(local.filter, ["metric.type=\"run.googleapis.com/instance/disk/utilizations\""])
  group_by_fields = ["metric.label.\"volume_name\""]
  primary_align   = "ALIGN_DELTA"
  primary_reduce  = "REDUCE_PERCENTILE_99"
}

module "instance_disk_usage" {
  source          = "../../widgets/xy"
  title           = "Instance disk usage"
  filter          = concat(local.filter, ["metric.type=\"run.googleapis.com/instance/disk/usage\""])
  group_by_fields = ["metric.label.\"volume_name\""]
  // Unlike the utilization metrics this one is a GAUGE distribution, which
  // ALIGN_DELTA cannot be applied to, so take the p99 across instances.
  primary_align  = "ALIGN_PERCENTILE_99"
  primary_reduce = "REDUCE_PERCENTILE_99"
}

module "instance_disk_allocation" {
  source          = "../../widgets/xy"
  title           = "Instance disk allocation"
  filter          = concat(local.filter, ["metric.type=\"run.googleapis.com/instance/disk/allocation_time\""])
  group_by_fields = ["metric.label.\"volume_name\""]
  // allocation_time counts GiB-seconds, so its rate is the number of GiB
  // currently allocated, summed across instances.
  primary_align  = "ALIGN_RATE"
  primary_reduce = "REDUCE_SUM"
  plot_type      = "STACKED_AREA"
}

module "memory_utilization" {
  source         = "../../widgets/xy"
  title          = "Memory utilization"
  filter         = concat(local.filter, ["metric.type=\"run.googleapis.com/container/memory/utilizations\""])
  primary_align  = "ALIGN_DELTA"
  primary_reduce = "REDUCE_PERCENTILE_99"
}

module "startup_latency" {
  source         = "../../widgets/xy"
  title          = "Startup latency"
  filter         = concat(local.filter, ["metric.type=\"run.googleapis.com/container/startup_latencies\""])
  primary_align  = "ALIGN_DELTA"
  primary_reduce = "REDUCE_PERCENTILE_99"
  plot_type      = "STACKED_BAR"
}

module "sent_bytes" {
  source         = "../../widgets/xy"
  title          = "Sent bytes"
  filter         = concat(local.filter, ["metric.type=\"run.googleapis.com/container/network/sent_bytes_count\""])
  primary_align  = "ALIGN_MEAN"
  primary_reduce = "REDUCE_NONE"
}

module "received_bytes" {
  source         = "../../widgets/xy"
  title          = "Received bytes"
  filter         = concat(local.filter, ["metric.type=\"run.googleapis.com/container/network/received_bytes_count\""])
  primary_align  = "ALIGN_MEAN"
  primary_reduce = "REDUCE_NONE"
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
      widget = module.cpu_utilization.widget,
    },
    {
      yPos   = 0,
      xPos   = local.col[1],
      height = local.unit,
      width  = local.unit,
      widget = module.memory_utilization.widget,
    },
    {
      yPos   = 0,
      xPos   = local.col[2],
      height = local.unit,
      width  = local.unit,
      widget = module.instance_count.widget,
    },
    {
      yPos   = local.unit,
      xPos   = local.col[0],
      height = local.unit,
      width  = local.unit,
      widget = module.startup_latency.widget,
    },
    {
      yPos   = local.unit,
      xPos   = local.col[1],
      height = local.unit,
      width  = local.unit,
      widget = module.sent_bytes.widget,
    },
    {
      yPos   = local.unit,
      xPos   = local.col[2],
      height = local.unit,
      width  = local.unit,
      widget = module.received_bytes.widget,
    },
    {
      yPos   = local.unit * 2,
      xPos   = local.col[0],
      height = local.unit,
      width  = local.unit,
      widget = module.disk_usage.widget,
    },
    {
      yPos   = local.unit * 2,
      xPos   = local.col[1],
      height = local.unit,
      width  = local.unit,
      widget = module.instance_disk_utilization.widget,
    },
    {
      yPos   = local.unit * 2,
      xPos   = local.col[2],
      height = local.unit,
      width  = local.unit,
      widget = module.instance_disk_usage.widget,
    },
    {
      yPos   = local.unit * 3,
      xPos   = local.col[0],
      height = local.unit,
      width  = local.unit,
      widget = module.instance_disk_allocation.widget,
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
