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
      thresholds        = var.thresholds
      timeshiftDuration = var.timeshift_duration
    }
  }
}
