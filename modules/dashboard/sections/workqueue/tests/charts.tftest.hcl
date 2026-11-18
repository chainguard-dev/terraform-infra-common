# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

# The pending-work chart stacks the queued backlog by priority class over the
# not-before band. Every dispatcher replica reports the whole backlog, so MIN
# and MAX reducers dedup the replicas' copies; the grouping must keep every
# label that splits the backlog, or the reducer would pick one class instead
# of the total.

variables {
  service_name    = "fixture"
  concurrent_work = 10
}

run "pending_work_by_priority_class" {
  command = plan

  assert {
    condition     = length([for tile in output.section : tile if try(tile.widget.title, "") == "Amount of work pending (queued by priority class vs not-before)"]) == 1
    error_message = "The section must have exactly one pending-work chart."
  }

  assert {
    condition = alltrue([
      for tile in output.section : (
        tile.widget.xyChart.dataSets[0].timeSeriesQuery.timeSeriesFilter.aggregation.groupByFields == tolist([
          "metric.label.\"service_name\"", "metric.label.\"queue_name\"", "metric.label.\"priority_class\"",
        ]) &&
        tile.widget.xyChart.dataSets[0].timeSeriesQuery.timeSeriesFilter.aggregation.perSeriesAligner == "ALIGN_MIN" &&
        tile.widget.xyChart.dataSets[0].timeSeriesQuery.timeSeriesFilter.aggregation.crossSeriesReducer == "REDUCE_MIN" &&
        strcontains(tile.widget.xyChart.dataSets[0].timeSeriesQuery.timeSeriesFilter.filter, "metric.type=\"prometheus.googleapis.com/workqueue_queued_keys_by_priority_class/gauge\"")
      ) if try(tile.widget.title, "") == "Amount of work pending (queued by priority class vs not-before)"
    ])
    error_message = "The queued band must read workqueue_queued_keys_by_priority_class and keep priority_class in its MIN grouping, so the reducer dedups replicas and never merges classes."
  }

  assert {
    condition = alltrue([
      for tile in output.section : (
        length(tile.widget.xyChart.dataSets) == 2 &&
        tile.widget.xyChart.dataSets[1].timeSeriesQuery.timeSeriesFilter.aggregation.groupByFields == tolist([
          "metric.label.\"service_name\"", "metric.label.\"queue_name\"",
        ]) &&
        tile.widget.xyChart.dataSets[1].timeSeriesQuery.timeSeriesFilter.aggregation.perSeriesAligner == "ALIGN_MAX" &&
        tile.widget.xyChart.dataSets[1].timeSeriesQuery.timeSeriesFilter.aggregation.crossSeriesReducer == "REDUCE_MAX" &&
        strcontains(tile.widget.xyChart.dataSets[1].timeSeriesQuery.timeSeriesFilter.filter, "metric.type=\"prometheus.googleapis.com/workqueue_notbefore_keys/gauge\"")
      ) if try(tile.widget.title, "") == "Amount of work pending (queued by priority class vs not-before)"
    ])
    error_message = "The not-before band must stay a single workqueue_notbefore_keys series per queue, deduped with MAX."
  }
}
