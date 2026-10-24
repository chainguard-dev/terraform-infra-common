terraform {
  required_version = ">= 1.9"
}

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
variable "collapsed" {
  type    = bool
  default = false
}
variable "notification_channels" {
  type = list(string)
}
variable "charts" {
  description = "The charts to include, laid out three to a row in the order of the default, which includes them all."
  type        = list(string)
  default     = ["cpu_utilization", "memory_utilization", "instance_count", "startup_latency", "sent_bytes", "received_bytes", "disk_usage"]

  validation {
    condition     = alltrue([for c in var.charts : contains(["cpu_utilization", "memory_utilization", "instance_count", "startup_latency", "sent_bytes", "received_bytes", "disk_usage"], c)])
    error_message = "Allowed values for 'charts' are cpu_utilization, memory_utilization, instance_count, startup_latency, sent_bytes, received_bytes and disk_usage."
  }
}

locals {
  filter = concat(var.filter, var.cloudrun_type == "job" ? [
    "resource.type=\"cloud_run_job\"",
    "resource.label.\"job_name\"=\"${var.cloudrun_name}\"",
    ] : [
    "resource.type=\"cloud_run_revision\"",
    "resource.label.\"service_name\"=\"${var.cloudrun_name}\"",
  ])
}

module "width" { source = "../width" }

// Jobs have no instance_count metric and no revisions; the closest equivalent
// is the number of task attempts running, per region.
module "instance_count" {
  source = "../../widgets/xy"
  title  = var.cloudrun_type == "job" ? "Running task attempts by region" : "Instance count + revisions"
  filter = concat(local.filter, [
    var.cloudrun_type == "job"
    ? "metric.type=\"run.googleapis.com/job/running_task_attempts\""
    : "metric.type=\"run.googleapis.com/container/instance_count\""
  ])
  group_by_fields = [var.cloudrun_type == "job" ? "resource.label.\"location\"" : "resource.label.\"revision_name\""]
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
  // The otel sidecar stamps the Cloud Run service or job name as service_name.
  filter = concat(var.filter, [
    "metric.label.\"service_name\"=\"${var.cloudrun_name}\"",
    "metric.type=\"prometheus.googleapis.com/disk_usage_bytes/gauge\"",
    "resource.type=\"prometheus_target\"",
  ])
  primary_align  = "ALIGN_MEAN"
  primary_reduce = "REDUCE_PERCENTILE_99"
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

  // Every chart in layout order; var.charts picks which of them to include.
  widgets = [
    { name = "cpu_utilization", widget = module.cpu_utilization.widget },
    { name = "memory_utilization", widget = module.memory_utilization.widget },
    { name = "instance_count", widget = module.instance_count.widget },
    { name = "startup_latency", widget = module.startup_latency.widget },
    { name = "sent_bytes", widget = module.sent_bytes.widget },
    { name = "received_bytes", widget = module.received_bytes.widget },
    { name = "disk_usage", widget = module.disk_usage.widget },
  ]
  included = [for w in local.widgets : w.widget if contains(var.charts, w.name)]

  tiles = [for i, w in local.included : {
    yPos   = local.unit * floor(i / local.columns)
    xPos   = local.unit * (i % local.columns)
    height = local.unit
    width  = local.unit
    widget = w
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
