locals { common_filter = ["resource.type=\"cloud_run_revision\""] }

module "logs" {
  source = "../widgets/logs"
  title  = "Service Logs"
  filter = local.common_filter
}

module "request_count" {
  source           = "../widgets/xy"
  title            = "Request count"
  filter           = concat(local.common_filter, ["metric.type=\"run.googleapis.com/request_count\""])
  group_by_fields  = ["metric.label.\"response_code_class\""]
  primary_align    = "ALIGN_RATE"
  primary_reduce   = "REDUCE_NONE"
  secondary_align  = "ALIGN_NONE"
  secondary_reduce = "REDUCE_SUM"
}

module "incoming_latency" {
  source = "../widgets/latency"
  title  = "Incoming request latency"
  filter = concat(local.common_filter, ["metric.type=\"run.googleapis.com/request_latencies\""])
}

module "instance_count" {
  source          = "../widgets/xy"
  title           = "Instance count + revisions"
  filter          = concat(local.common_filter, ["metric.type=\"run.googleapis.com/container/instance_count\""])
  group_by_fields = ["resource.label.\"revision_name\""]
  primary_align   = "ALIGN_MEAN"
  primary_reduce  = "REDUCE_SUM"
  plot_type       = "STACKED_AREA"
}

module "cpu_utilization" {
  source         = "../widgets/xy"
  title          = "CPU utilization"
  filter         = concat(local.common_filter, ["metric.type=\"run.googleapis.com/container/cpu/utilizations\""])
  primary_align  = "ALIGN_DELTA"
  primary_reduce = "REDUCE_MEAN"
}

module "memory_utilization" {
  source         = "../widgets/xy"
  title          = "Memory utilization"
  filter         = concat(local.common_filter, ["metric.type=\"run.googleapis.com/container/memory/utilizations\""])
  primary_align  = "ALIGN_DELTA"
  primary_reduce = "REDUCE_MEAN"
}

module "startup_latency" {
  source         = "../widgets/xy"
  title          = "Startup latency"
  filter         = concat(local.common_filter, ["metric.type=\"run.googleapis.com/container/startup_latencies\""])
  primary_align  = "ALIGN_DELTA"
  primary_reduce = "REDUCE_MEAN"
}

module "sent_bytes" {
  source         = "../widgets/xy"
  title          = "Sent bytes"
  filter         = concat(local.common_filter, ["metric.type=\"run.googleapis.com/container/network/sent_bytes_count\""])
  primary_align  = "ALIGN_MEAN"
  primary_reduce = "REDUCE_NONE"
}

module "received_bytes" {
  source         = "../widgets/xy"
  title          = "Received bytes"
  filter         = concat(local.common_filter, ["metric.type=\"run.googleapis.com/container/network/received_bytes_count\""])
  primary_align  = "ALIGN_MEAN"
  primary_reduce = "REDUCE_NONE"
}

resource "google_monitoring_dashboard" "dashboard" {
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
      widgets = [
        module.logs.widget,
        module.request_count.widget,
        module.incoming_latency.widget,
        module.instance_count.widget,
        module.cpu_utilization.widget,
        module.memory_utilization.widget,
        module.startup_latency.widget,
        module.sent_bytes.widget,
        module.received_bytes.widget,
      ]
    }
  })
}
