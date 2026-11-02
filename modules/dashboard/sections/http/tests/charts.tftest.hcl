# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

# The section's charts, one row per direction. Its first tile is the
# collapsible header, so a chart's tile is one past its index.

variables {
  title        = "HTTP"
  filter       = []
  service_name = "fixture"
}

run "service_charts_both_directions" {
  command = plan
  assert {
    condition = [for tile in slice(output.section, 1, length(output.section)) : [tile.widget.title, tile.yPos, tile.xPos]] == [
      ["Incoming request count", 0, 0], ["Incoming request latency", 0, 4], ["Incoming request failure rate", 0, 8],
      ["Outbound request count", 6, 0], ["Outbound request latency", 6, 4], ["Outbound request failure rate", 6, 8],
    ]
    error_message = "A service charts incoming requests on the first row and outbound calls on the second."
  }
  # A failing code's share of all requests: the numerator splits by code, the
  # denominator sums every request.
  assert {
    condition = alltrue([
      for i, code in { 3 = "metric.label.\"response_code\"", 6 = "metric.label.\"code\"" } :
      output.section[i].widget.xyChart.dataSets[0].timeSeriesQuery.timeSeriesFilterRatio.numerator.aggregation.groupByFields == tolist([code]) &&
      length(output.section[i].widget.xyChart.dataSets[0].timeSeriesQuery.timeSeriesFilterRatio.denominator.aggregation.groupByFields) == 0
    ])
    error_message = "Each failure rate must split failures by code over all requests."
  }
}

run "job_charts_outbound_only" {
  command = plan
  variables {
    cloudrun_type = "job"
  }
  assert {
    condition = [for tile in slice(output.section, 1, length(output.section)) : [tile.widget.title, tile.yPos, tile.xPos]] == [
      ["Outbound request count", 0, 0], ["Outbound request latency", 0, 4], ["Outbound request failure rate", 0, 8],
    ]
    error_message = "A job serves no requests, so it charts only outbound calls, from the first row."
  }
}
