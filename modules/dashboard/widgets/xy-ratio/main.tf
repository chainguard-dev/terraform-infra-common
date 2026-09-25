variable "title" { type = string }
variable "legend" {
  type    = string
  default = ""
}
variable "numerator_group_by_fields" {
  type    = list(string)
  default = []
}
variable "denominator_group_by_fields" {
  type    = list(string)
  default = []
}
variable "numerator_filter" { type = list(string) }
variable "denominator_filter" { type = list(string) }
variable "plot_type" {
  type    = string
  default = "LINE"
}
variable "alignment_period" {
  type    = string
  default = "60s"
}
variable "numerator_align" {
  type    = string
  default = "ALIGN_RATE"
}
variable "numerator_reduce" {
  type    = string
  default = "REDUCE_SUM"
}
variable "denominator_align" {
  type    = string
  default = "ALIGN_RATE"
}
variable "denominator_reduce" {
  type    = string
  default = "REDUCE_SUM"
}
variable "thresholds" {
  type    = list(number)
  default = []
}

// https://cloud.google.com/monitoring/api/ref_v3/rest/v1/projects.dashboards#XyChart
output "widget" {
  value = {
    title = var.title
    xyChart = {
      chartOptions = { mode = "COLOR" }
      dataSets = [{
        minAlignmentPeriod = var.alignment_period
        plotType           = var.plot_type
        targetAxis         = "Y1"
        legendTemplate     = var.legend
        timeSeriesQuery = {
          // https://cloud.google.com/monitoring/api/ref_v3/rest/v1/projects.dashboards#timeseriesfilterratio
          timeSeriesFilterRatio = {
            numerator = {
              filter = join("\n", var.numerator_filter)
              aggregation = {
                alignmentPeriod    = var.alignment_period
                perSeriesAligner   = var.numerator_align
                crossSeriesReducer = var.numerator_reduce
                groupByFields      = var.numerator_group_by_fields
              }
            }
            denominator = {
              filter = join("\n", var.denominator_filter)
              aggregation = {
                alignmentPeriod    = var.alignment_period
                perSeriesAligner   = var.denominator_align
                crossSeriesReducer = var.denominator_reduce
                groupByFields      = var.denominator_group_by_fields
              }
            }
          }
        }
      }]
      thresholds = [
        for threshold in var.thresholds : {
          value      = threshold
          targetAxis = "Y1"
        }
      ]
      yAxis = {
        label = "y1Axis"
        scale = "LINEAR"
      }
    }
  }
}
