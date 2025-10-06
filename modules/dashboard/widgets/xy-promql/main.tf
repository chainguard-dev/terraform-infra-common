/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

output "widget" {
  value = {
    title = var.title
    xyChart = {
      chartOptions = { mode = "COLOR" }
      dataSets = [{
        plotType   = var.plot_type
        targetAxis = "Y1"
        timeSeriesQuery = {
          prometheusQuery = var.promql_query
        }
      }]
      yAxis = {
        label = "y1Axis"
        scale = "LINEAR"
      }
      thresholds = [
        for threshold in var.thresholds : {
          value      = threshold
          targetAxis = "Y1"
        }
      ],
      timeshiftDuration = var.timeshift_duration
    }
  }
}
