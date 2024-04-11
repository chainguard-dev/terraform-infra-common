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

locals {
  slo_threshold_friendly = "${var.slo_threshold * 100}%"
}

// Create an alert policy based on the service's SLO threshold,
// as measured by the uptime check success percent over a rolling window.
resource "google_monitoring_alert_policy" "slo_alert" {
  count   = var.enable_slo_alert ? 1 : 0
  project = var.project_id

  # In the absence of data, incident will auto-close in 7 days
  alert_strategy {
    auto_close = "${7 * 24 * 60 * 60}s"
  }
  combiner = "OR"

  conditions {
    condition_threshold {
      aggregations {
        alignment_period     = "${24 * 60 * 60}s" # 1 day
        cross_series_reducer = "REDUCE_MIN"
        per_series_aligner   = "ALIGN_FRACTION_TRUE"
      }

      comparison = "COMPARISON_LT"
      duration   = "0s"
      filter     = <<-EOT
        metric.type="monitoring.googleapis.com/uptime_check/check_passed"
        resource.type="uptime_url"
        metric.label.check_id=starts_with("${local.uptime_check_name}")
      EOT

      threshold_value = var.slo_threshold
      trigger {
        count   = 1
      }
    }

    display_name = "${local.uptime_check_name} uptime less than ${local.slo_threshold_friendly}"
  }

  display_name = "${local.uptime_check_name} uptime less than ${local.slo_threshold_friendly} alert"
  enabled      = true

  documentation {
    content = "${local.uptime_check_name} has fallen below ${local.slo_threshold_friendly} over the past day."
  }

  notification_channels = var.slo_notification_channels
}
