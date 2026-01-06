# Alert policies for monitoring log ingestion health
resource "google_monitoring_alert_policy" "log_ingestion" {
  for_each = var.enable_monitoring ? var.tables : {}

  project      = var.project_id
  display_name = "${var.name} - ${each.key} - No logs ingested"

  documentation {
    content = "No logs have been ingested to table ${each.key} in the last ${var.alert_threshold_minutes} minutes."
  }

  combiner = "OR"

  conditions {
    display_name = "No successful log writes in ${var.alert_threshold_minutes} minutes"

    condition_threshold {
      filter          = <<-EOT
        resource.type = "logging_sink"
        AND resource.labels.name = "${google_logging_project_sink.sinks[each.key].name}"
        AND metric.type = "logging.googleapis.com/exports/log_entry_count"
      EOT
      duration        = "${var.alert_threshold_minutes * 60}s"
      comparison      = "COMPARISON_LT"
      threshold_value = 1

      aggregations {
        alignment_period   = "300s"
        per_series_aligner = "ALIGN_RATE"
      }
    }
  }

  alert_strategy {
    auto_close = "${var.alert_auto_close_days * 24 * 60 * 60}s"
  }

  notification_channels = var.notification_channels

  enabled = true
}
