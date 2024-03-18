resource "google_monitoring_alert_policy" "pubsub_dead_letter_queue_messages" {
  alert_strategy {
    auto_close = "3600s" // 1 hour
  }

  combiner = "OR"

  conditions {
    condition_threshold {
      aggregations {
        alignment_period   = "60s"
        per_series_aligner = "ALIGN_MAX"
      }

      comparison = "COMPARISON_GT"
      duration   = "0s"
      filter     = "metric.type=\"pubsub.googleapis.com/topic/send_request_count\" resource.type=\"pubsub_topic\" metadata.system_labels.\"name\"=\"${google_pubsub_topic.dead-letter.name}\""

      trigger {
        count = "1"
      }

      // TODO: make configurable later
      threshold_value = 1
    }

    display_name = "${var.name}-${var.private-service.region}: dead-letter queue messages above 1"
  }
  display_name = "${var.name}-${var.private-service.region}: dead-letter queue messages above 1"

  enabled               = "true"
  notification_channels = var.notification_channels
}
