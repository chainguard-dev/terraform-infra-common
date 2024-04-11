variable "title" { type = string }
variable "group_by_fields" { default = [] }
variable "filter" { type = list(string) }
variable "band" {
  type    = number
  default = 99
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
        timeSeriesQuery = {
          timeSeriesFilter = {
            aggregation = {
              alignmentPeriod    = "60s"
              perSeriesAligner   = "ALIGN_DELTA"
              crossSeriesReducer = "REDUCE_PERCENTILE_${var.band}"
              groupByFields      = var.group_by_fields
            }
            filter = join("\n", var.filter)
          }
        }
      }]
      timeshiftDuration = "0s"
      yAxis = {
        label = "y1Axis"
        scale = "LINEAR"
      }
    }
  }
}
