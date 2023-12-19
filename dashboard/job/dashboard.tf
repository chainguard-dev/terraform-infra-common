locals { common_filter = ["resource.type=\"cloud_run_job\""] }

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
      widgets = [
        module.logs.tile,
        module.cpu_utilization.tile,
        module.memory_utilization.tile,
        module.startup_latency.tile,
        module.sent_bytes.tile,
        module.received_bytes.tile,
      ]
    }
  })
}
