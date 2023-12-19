locals {
  common_filter = ["resource.type=\"cloud_run_revision\""]
}

module "alert" {
  for_each   = var.alert_policies
  source     = "../tiles/alert"
  title      = "Alert: ${each.key}"
  alert_name = each.value.id
}

module "logs" {
  source = "../tiles/logs"
  title  = "Service Logs"
  filter = local.common_filter
}

module "request_count" {
  source           = "../tiles/xy"
  title            = "Request count"
  filter           = concat(local.common_filter, ["metric.type=\"run.googleapis.com/request_count\""])
  group_by_fields  = ["metric.label.\"response_code_class\""]
  primary_align    = "ALIGN_RATE"
  primary_reduce   = "REDUCE_NONE"
  secondary_align  = "ALIGN_NONE"
  secondary_reduce = "REDUCE_SUM"
}

module "incoming_latency" {
  source = "../tiles/latency"
  title  = "Incoming request latency"
  filter = concat(local.common_filter, ["metric.type=\"run.googleapis.com/request_latencies\""])
}

module "instance_count" {
  source          = "../tiles/xy"
  title           = "Instance count + revisions"
  filter          = concat(local.common_filter, ["metric.type=\"run.googleapis.com/container/instance_count\""])
  group_by_fields = ["resource.label.\"revision_name\""]
  primary_align   = "ALIGN_MEAN"
  primary_reduce  = "REDUCE_SUM"
  plot_type       = "STACKED_AREA"
}

module "cpu_utilization" {
  source         = "../tiles/xy"
  title          = "CPU utilization"
  filter         = concat(local.common_filter, ["metric.type=\"run.googleapis.com/container/cpu/utilizations\""])
  primary_align  = "ALIGN_DELTA"
  primary_reduce = "REDUCE_MEAN"
}

module "memory_utilization" {
  source         = "../tiles/xy"
  title          = "Memory utilization"
  filter         = concat(local.common_filter, ["metric.type=\"run.googleapis.com/container/memory/utilizations\""])
  primary_align  = "ALIGN_DELTA"
  primary_reduce = "REDUCE_MEAN"
}

module "startup_latency" {
  source         = "../tiles/xy"
  title          = "Startup latency"
  filter         = concat(local.common_filter, ["metric.type=\"run.googleapis.com/container/startup_latencies\""])
  primary_align  = "ALIGN_DELTA"
  primary_reduce = "REDUCE_MEAN"
}

module "sent_bytes" {
  source         = "../tiles/xy"
  title          = "Sent bytes"
  filter         = concat(local.common_filter, ["metric.type=\"run.googleapis.com/container/network/sent_bytes_count\""])
  primary_align  = "ALIGN_MEAN"
  primary_reduce = "REDUCE_NONE"
}

module "received_bytes" {
  source         = "../tiles/xy"
  title          = "Received bytes"
  filter         = concat(local.common_filter, ["metric.type=\"run.googleapis.com/container/network/received_bytes_count\""])
  primary_align  = "ALIGN_MEAN"
  primary_reduce = "REDUCE_NONE"
}

resource "google_monitoring_dashboard" "dashboard" {
  project = var.project_id

  dashboard_json = jsonencode({
    displayName = "Cloud Run Service: ${var.service_name}"
    dashboardFilters = [{
      filterType  = "RESOURCE_LABEL"
      stringValue = var.service_name
      labelKey    = "service_name"
    }]
    //  https://cloud.google.com/monitoring/api/ref_v3/rest/v1/projects.dashboards#GridLayout
    gridLayout = {
      columns = 3
      widgets = concat(
        [for k in sort(keys(var.alert_policies)) : module.alert[k].tile],
        [
          module.logs.tile,
          module.request_count.tile,
          module.incoming_latency.tile,
          module.instance_count.tile,
          module.cpu_utilization.tile,
          module.memory_utilization.tile,
          module.startup_latency.tile,
          module.sent_bytes.tile,
          module.received_bytes.tile,
        ],
      )
    }
  })
}

locals {
  parts        = split("/", resource.google_monitoring_dashboard.dashboard.id)
  dashboard_id = local.parts[length(local.parts) - 1]
}

output "url" {
  value = "https://console.cloud.google.com/monitoring/dashboards/builder/${local.dashboard_id};duration=P1D?project=${var.project_id}"
}
