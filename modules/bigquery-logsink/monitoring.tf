# Alert policies for monitoring log ingestion health
resource "google_monitoring_alert_policy" "log_ingestion" {
  for_each = var.enable_monitoring ? var.sinks : {}

  project      = var.project_id
  display_name = "${var.name} - ${each.key} - No logs ingested"

  documentation {
    content = <<-EOT
      Log sink `${google_logging_project_sink.sinks[each.key].name}` exported no log entries to BigQuery dataset `${google_bigquery_dataset.this.dataset_id}` in the last ${var.alert_threshold_minutes} minutes. Queries and reports over that dataset are missing everything since then.

      Troubleshooting:
      1. Run the sink's filter in Logs Explorer (link below). No matches: the service that writes these logs is down, idle, or changed its log format or resource type. Check that service and escalate to its owners.
      2. Matches but no exports: open the Log Router (link below) and check the sink for errors. A likely cause is the sink's writer identity losing `roles/bigquery.dataEditor` on the dataset; the `logging.googleapis.com/exports/error_count` metric counts failed exports.
      3. The incident closes once the sink exports again.
    EOT

    links {
      display_name = "Logs Explorer: sink filter"
      url          = "https://console.cloud.google.com/logs/query;query=${replace(urlencode(each.value.log_filter), "+", "%20")}?project=${var.project_id}"
    }

    links {
      display_name = "Log Router"
      url          = "https://console.cloud.google.com/logs/router?project=${var.project_id}"
    }
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

      # The metric writes no point for a window without exports, so a stopped
      # sink leaves nothing to compare. Count missing data as met.
      evaluation_missing_data = "EVALUATION_MISSING_DATA_ACTIVE"

      aggregations {
        alignment_period = "300s"
        # Entries per window. ALIGN_RATE is per second, so < 1 would mean
        # fewer than 300 entries per window.
        per_series_aligner = "ALIGN_SUM"
      }
    }
  }

  alert_strategy {
    auto_close = "${var.alert_auto_close_days * 24 * 60 * 60}s"
  }

  notification_channels = var.notification_channels

  enabled = true
}
