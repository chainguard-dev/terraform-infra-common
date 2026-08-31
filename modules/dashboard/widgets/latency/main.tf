variable "title" { type = string }
variable "group_by_fields" { default = [] }
variable "filter" { type = list(string) }
variable "band" {
  type    = number
  default = 99
}
variable "unit" {
  description = "Unit for chart values, using Cloud Monitoring's unit format."
  type        = string
  default     = ""
}

// https://cloud.google.com/monitoring/api/ref_v3/rest/v1/projects.dashboards#XyChart
output "widget" {
  value = {
    title = var.title
    xyChart = {
      chartOptions = { mode = "COLOR" }
      dataSets = [{
        minAlignmentPeriod = "60s"
        plotType           = "LINE"
        targetAxis         = "Y1"
        timeSeriesQuery = merge({
          timeSeriesFilter = {
            aggregation = {
              alignmentPeriod    = "60s"
              perSeriesAligner   = "ALIGN_DELTA"
              crossSeriesReducer = "REDUCE_PERCENTILE_${var.band}"
              groupByFields      = var.group_by_fields
            }
            filter = join("\n", var.filter)
          }
        }, var.unit != "" ? { unitOverride = var.unit } : {})
      }]
      timeshiftDuration = "0s"
      yAxis = {
        label = "y1Axis"
        scale = "LINEAR"
      }
    }
  }
}
