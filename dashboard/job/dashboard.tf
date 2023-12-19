locals {
  common_filter = ["resource.type=\"cloud_run_job\""]
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
    displayName = "Cloud Run Job: ${var.job_name}"
    dashboardFilters = [{
      filterType  = "RESOURCE_LABEL"
      stringValue = var.job_name
      labelKey    = "job_name"
    }]
    //  https://cloud.google.com/monitoring/api/ref_v3/rest/v1/projects.dashboards#GridLayout
    gridLayout = {
      columns = 3
      widgets = concat(
        [for k in sort(keys(var.alert_policies)) : module.alert[k].tile],
        [
          module.logs.tile,
          module.cpu_utilization.tile,
          module.memory_utilization.tile,
          module.startup_latency.tile,
          module.sent_bytes.tile,
          module.received_bytes.tile,

          // These also work:
          //{ text = {
          //  content = "_Created on ${timestamp()}_",
          //  format  = "MARKDOWN"
          //} },
          //{ blank = {} },

          // Only allowed in mosaicLayout, where we manage rows/columns ourselves :(
          // { collapsibleGroup = { collapsed = true } },

          // NB: Sometimes updating the dashboard fails due to:
          // https://github.com/hashicorp/terraform-provider-google/issues/16439
          // When this happens, terraform destroy and apply again.
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
