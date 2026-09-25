/*
Copyright 2023 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Create an alert policy based on the uptime check.
resource "google_monitoring_alert_policy" "uptime_alert" {
  count   = var.enable_alert ? 1 : 0
  project = var.project_id

  severity = "CRITICAL"

  # In the absence of data, incident will auto-close in 7 days
  alert_strategy {
    auto_close = "${7 * 24 * 60 * 60}s"
  }
  combiner = "OR"

  conditions {
    condition_threshold {
      aggregations {
        alignment_period     = var.period
        cross_series_reducer = "REDUCE_COUNT_FALSE"
        group_by_fields      = ["resource.*"]
        per_series_aligner   = "ALIGN_NEXT_OLDER"
      }

      comparison = "COMPARISON_GT"
      duration   = var.uptime_alert_duration
      // With service_agent_auth the uptime check monitors the Cloud Run
      // service, so check_passed is emitted under cloud_run_revision
      // rather than uptime_url.
      filter = <<-EOT
        metric.type="monitoring.googleapis.com/uptime_check/check_passed"
        resource.type="${var.service_agent_auth ? "cloud_run_revision" : "uptime_url"}"
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

    dynamic "links" {
      for_each = var.alert_links
      content {
        display_name = links.value.display_name
        url          = links.value.url
      }
    }
  }

  notification_channels = var.notification_channels
}
