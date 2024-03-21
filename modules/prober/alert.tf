/*
Copyright 2023 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Create an alert policy based on the uptime check.
resource "google_monitoring_alert_policy" "uptime_alert" {
  count   = var.enable_alert ? 1 : 0
  project = var.project_id

  # In the absence of data, incident will auto-close in 7 days
  alert_strategy {
    auto_close = "${7 * 24 * 60 * 60}s"
  }
  combiner = "OR"

  conditions {
    condition_threshold {
      aggregations {
        alignment_period     = "300s"
        cross_series_reducer = "REDUCE_COUNT_FALSE"
        group_by_fields      = ["resource.*"]
        per_series_aligner   = "ALIGN_NEXT_OLDER"
      }

      comparison = "COMPARISON_GT"
      duration   = "60s"
      filter     = <<-EOT
        metric.type="monitoring.googleapis.com/uptime_check/check_passed"
        resource.type="uptime_url"
        metric.label.check_id="${local.uptime_check_id}"
      EOT

      // TODO(jason): Make these configurable.
      threshold_value = 2
      trigger {
        count   = 1
        percent = 0
      }
    }

    display_name = "${local.uptime_check_name} probe failure"
  }

  display_name = "${local.uptime_check_name} prober failed alert"
  enabled      = true

  documentation {
    content = var.alert_description
  }

  notification_channels = var.notification_channels
}
