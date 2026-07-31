variable "title" { type = string }
variable "group_by_fields" { default = [] }
variable "filter" { type = list(string) }
variable "plot_type" { default = "LINE" }
variable "alignment_period" { default = "60s" }
variable "primary_align" { default = "ALIGN_RATE" }
variable "primary_reduce" { default = "REDUCE_NONE" }
variable "secondary_align" { default = "" }
variable "secondary_reduce" { default = "" }
variable "thresholds" { default = [] }
variable "legend" {
  description = "Legend template for the primary dataset."
  type        = string
  default     = ""
}
variable "datasets" {
  description = "Additional datasets to plot on the same chart alongside the primary one."
  type = list(object({
    filter          = list(string)
    group_by_fields = optional(list(string), [])
    plot_type       = optional(string, "LINE")
    align           = optional(string, "ALIGN_RATE")
    reduce          = optional(string, "REDUCE_NONE")
    legend          = optional(string, "")
  }))
  default = []
}

locals {
  use_secondary    = var.secondary_align != "" || var.secondary_reduce != ""
  secondary_align  = var.secondary_align != "" ? var.secondary_align : "ALIGN_NONE"
  secondary_reduce = var.secondary_reduce != "" ? var.secondary_reduce : "REDUCE_NONE"
}

// https://cloud.google.com/monitoring/api/ref_v3/rest/v1/projects.dashboards#XyChart
output "widget" {
  value = {
    title = var.title
    xyChart = {
      chartOptions = { mode = "COLOR" }
      dataSets = concat([{
        minAlignmentPeriod = var.alignment_period
        plotType           = var.plot_type
        targetAxis         = "Y1"
        legendTemplate     = var.legend != "" ? var.legend : null
        timeSeriesQuery = {
          timeSeriesFilter = {
            aggregation = {
              alignmentPeriod    = var.alignment_period
              perSeriesAligner   = var.primary_align
              crossSeriesReducer = var.primary_reduce
              groupByFields      = var.group_by_fields
            }
            filter = join("\n", var.filter)
            secondaryAggregation = local.use_secondary ? {
              alignmentPeriod    = var.alignment_period
              perSeriesAligner   = var.secondary_align
              crossSeriesReducer = var.secondary_reduce
              groupByFields      = var.group_by_fields
            } : null
          }
        }
        }], [for ds in var.datasets : {
        minAlignmentPeriod = var.alignment_period
        plotType           = ds.plot_type
        targetAxis         = "Y1"
        legendTemplate     = ds.legend != "" ? ds.legend : null
        timeSeriesQuery = {
          timeSeriesFilter = {
            aggregation = {
              alignmentPeriod    = var.alignment_period
              perSeriesAligner   = ds.align
              crossSeriesReducer = ds.reduce
              groupByFields      = ds.group_by_fields
            }
            filter = join("\n", ds.filter)
          }
        }
      }])
      thresholds = [
        for threshold in var.thresholds : {
          value      = threshold
          targetAxis = "Y1"
        }
      ],
      timeshiftDuration = "0s"
      yAxis = {
        label = "y1Axis"
        scale = "LINEAR"
      }
    }
  }
}
