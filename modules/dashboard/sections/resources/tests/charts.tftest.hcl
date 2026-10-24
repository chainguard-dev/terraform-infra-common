# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

# The section's charts, laid out three to a row. Its first tile is the
# collapsible header, so a chart's tile is one past its index.

variables {
  title                 = "Resources"
  filter                = []
  cloudrun_name         = "fixture"
  notification_channels = []
}

# Every chart by default, in the layout every user had before charts was
# selectable.
run "default_includes_every_chart" {
  command = plan
  assert {
    condition = [for tile in slice(output.section, 1, length(output.section)) : [tile.widget.title, tile.yPos, tile.xPos]] == [
      ["CPU utilization", 0, 0], ["Memory utilization", 0, 4], ["Instance count + revisions", 0, 8],
      ["Startup latency", 4, 0], ["Sent bytes", 4, 4], ["Received bytes", 4, 8],
      ["Disk usage", 8, 0],
    ]
    error_message = "The default section holds all seven charts in their usual places."
  }
  assert {
    condition     = output.section[0].height == 12
    error_message = "The default section's header spans its three rows."
  }
}

# A selection keeps the default's order, whatever order it names them in,
# and packs them from the first row.
run "selection_packs_in_default_order" {
  command = plan
  variables {
    charts = ["instance_count", "cpu_utilization", "memory_utilization"]
  }
  assert {
    condition = [for tile in slice(output.section, 1, length(output.section)) : [tile.widget.title, tile.yPos, tile.xPos]] == [
      ["CPU utilization", 0, 0], ["Memory utilization", 0, 4], ["Instance count + revisions", 0, 8],
    ]
    error_message = "A selection of three charts fills one row in the default's order."
  }
  assert {
    condition     = output.section[0].height == 4
    error_message = "A one-row section's header spans one row."
  }
}

run "unknown_chart_rejected" {
  command = plan
  variables {
    charts = ["cpu_utilization", "latency"]
  }
  expect_failures = [var.charts]
}
