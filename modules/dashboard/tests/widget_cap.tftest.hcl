# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

# Plan-only guard on the 50-widget-per-dashboard limit that Cloud Monitoring
# enforces. Every dashboard in the repo is rendered through this module, so the
# precondition here is the single choke point: a dashboard that would be rejected
# by the Monitoring API (">50 widgets") fails at plan time instead, and this
# suite proves the guard fires on the leaf-widget count while ignoring the
# collapsibleGroup section headers (which GCP does not count).
#
# The google provider is mocked so the plan stays fully offline: no credentials,
# no state, no API calls. Preconditions are still evaluated during plan.

mock_provider "google" {}

# 50 leaf widgets (plus a section header) is exactly at the limit and must pass.
run "at_the_cap_passes" {
  command = plan

  variables {
    object = {
      displayName = "fixture-at-cap"
      mosaicLayout = {
        columns = 48
        tiles = concat(
          [{ widget = { title = "section", collapsibleGroup = { collapsed = false } } }],
          [for i in range(50) : { widget = { title = "w${i}", xyChart = {} } }],
        )
      }
    }
  }

  assert {
    condition     = output.json != ""
    error_message = "expected the dashboard JSON to render for a 50-widget dashboard"
  }
}

# 51 leaf widgets is over the limit and must fail the precondition.
run "over_the_cap_fails" {
  command = plan

  variables {
    object = {
      displayName = "fixture-over-cap"
      mosaicLayout = {
        columns = 48
        tiles   = [for i in range(51) : { widget = { title = "w${i}", xyChart = {} } }]
      }
    }
  }

  expect_failures = [
    google_monitoring_dashboard.dashboard,
  ]
}

# collapsibleGroup section headers are not counted: 50 leaves + 6 headers (56
# tiles) is still under the cap and must pass. This pins the exclusion so a
# future refactor that starts counting headers would fail here.
run "section_headers_do_not_count" {
  command = plan

  variables {
    object = {
      displayName = "fixture-with-headers"
      mosaicLayout = {
        columns = 48
        tiles = concat(
          [for g in range(6) : { widget = { title = "section${g}", collapsibleGroup = { collapsed = false } } }],
          [for i in range(50) : { widget = { title = "w${i}", xyChart = {} } }],
        )
      }
    }
  }

  assert {
    condition     = output.json != ""
    error_message = "section headers must not count toward the 50-widget limit"
  }
}

# gridLayout dashboards (e.g. ecosystems/rebuilder) count their widgets too, not
# just mosaicLayout tiles: 50 grid widgets is at the cap and must pass.
run "grid_at_the_cap_passes" {
  command = plan

  variables {
    object = {
      displayName = "fixture-grid-at-cap"
      gridLayout = {
        columns = 2
        widgets = [for i in range(50) : { title = "w${i}", xyChart = {} }]
      }
    }
  }

  assert {
    condition     = output.json != ""
    error_message = "expected a 50-widget gridLayout dashboard to render"
  }
}

# 51 grid widgets is over the cap and must fail the precondition — this is the
# case a mosaic-only count would have missed.
run "grid_over_the_cap_fails" {
  command = plan

  variables {
    object = {
      displayName = "fixture-grid-over-cap"
      gridLayout = {
        columns = 2
        widgets = [for i in range(51) : { title = "w${i}", xyChart = {} }]
      }
    }
  }

  expect_failures = [
    google_monitoring_dashboard.dashboard,
  ]
}
