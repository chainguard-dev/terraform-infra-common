variable "object" {
  description = "Object to encode into JSON"
}

locals {
  json = replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(
    jsonencode(var.object),
    "\"collapsed\":false", ""),
    ",\"xPos\":0", ""),
    ",\"yPos\":0", ""),
    ",\"thresholds\":[]", ""),
    ",\"crossSeriesReducer\":\"REDUCE_NONE\"", ""),
    ",\"perSeriesAligner\":\"ALIGN_NONE\"", ""),
    "\"dashboardFilters\":[],", ""),
    ",\"groupByFields\":[]", ""),
    ",\"secondaryAggregation\":null", ""),
    "\"secondaryAggregation\":null,", ""),
    "\"secondaryAggregation\":null", ""),
    ",\"groupByFields\":null", ""),
  "\"groupByFields\":null,", "")
}

resource "google_monitoring_dashboard" "dashboard" {
  dashboard_json = local.json
}

output "json" {
  value = local.json
}
