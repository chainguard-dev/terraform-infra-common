resource "google_monitoring_alert_policy" "dead_letter_queue" {
  count = var.max-retry > 0 && var.enable_dead_letter_alerting ? 1 : 0

  project      = var.project_id
  display_name = "Workqueue dead-lettered keys ${var.name}"
  combiner     = "OR"
  severity     = "ERROR"

  alert_strategy {
    auto_close = "3600s"
  }

  conditions {
    display_name = "Workqueue dead-letter queue ${var.name}"

    condition_threshold {
      comparison      = "COMPARISON_GT"
      threshold_value = 1
      duration        = "0s"

      filter = <<EOT
        resource.type = "prometheus_target"
        AND metric.type = "prometheus.googleapis.com/workqueue_dead_lettered_keys/gauge"
        AND metric.label."service_name" = "${var.name}-dsp"
      EOT

      aggregations {
        alignment_period     = "60s"
        cross_series_reducer = "REDUCE_NONE"
        per_series_aligner   = "ALIGN_MAX"
      }

      trigger {
        count = 1
      }
    }
  }

  documentation {
    subject = "Workqueue ${var.name} has dead-lettered keys"
    content = "${var.name}-dsp has dead-lettered keys above threshold. Investigate and drain the dead-letter queue."
  }

  notification_channels = var.notification_channels
}
