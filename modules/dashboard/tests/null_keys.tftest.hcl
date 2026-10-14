# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

# The Monitoring API drops null keys, and the provider's diff suppression only
# ignores keys missing from config, so a null left in the JSON shows the whole
# dashboard as changed on every plan.

mock_provider "google" {}

run "null_legend_template_is_stripped" {
  command = plan

  variables {
    object = {
      displayName = "fixture-legends"
      mosaicLayout = {
        columns = 48
        tiles = [{
          widget = {
            title = "w"
            xyChart = {
              dataSets = [
                { legendTemplate = null, plotType = "LINE" },
                { legendTemplate = "p50", plotType = "LINE" },
              ]
            }
          }
        }]
      }
    }
  }

  assert {
    condition     = !strcontains(output.json, "\"legendTemplate\":null")
    error_message = "a null legendTemplate must not reach the dashboard JSON"
  }

  assert {
    condition     = strcontains(output.json, "\"legendTemplate\":\"p50\"")
    error_message = "a set legendTemplate must be kept"
  }
}
