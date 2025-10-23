locals {
  region_filter = "one_of(${join(", ", formatlist("%q", var.regions))})"
}

resource "google_monitoring_custom_service" "this" {
  service_id = "${lower(var.service_type)}-${lower(var.service_name)}"
  project    = var.project_id
}

resource "google_monitoring_slo" "success_cr" {
  service      = google_monitoring_custom_service.this.service_id
  slo_id       = "${var.service_name}-success-multi-region"
  display_name = "${var.service_name} - Multi-region Success Cloud Run SLO"
  project      = var.project_id

  goal                = var.slo.success.multi_region_goal
  rolling_period_days = 30

  request_based_sli {
    good_total_ratio {
      # Total requests
      total_service_filter = join(" AND ", [
        "metric.type=\"run.googleapis.com/request_count\"",
        "resource.type=\"cloud_run_revision\"",
        "resource.label.service_name=\"${var.service_name}\""
      ])

      # Bad requests
      bad_service_filter = join(" AND ", [
        "metric.type=\"run.googleapis.com/request_count\"",
        "resource.type=\"cloud_run_revision\"",
        "resource.label.service_name=\"${var.service_name}\"",
        "metric.label.response_code_class=\"5xx\""
      ])
    }
  }
}

resource "google_monitoring_slo" "success_cr_per_region" {
  for_each = toset(var.regions)

  service      = google_monitoring_custom_service.this.service_id
  slo_id       = "${var.service_name}-success-${each.value}"
  display_name = "${var.service_name} - ${each.key} Success Cloud Run SLO"
  project      = var.project_id

  goal                = var.slo.success.per_region_goal
  rolling_period_days = 30

  request_based_sli {
    good_total_ratio {
      # Total requests
      total_service_filter = join(" AND ", [
        "metric.type=\"run.googleapis.com/request_count\"",
        "resource.type=\"cloud_run_revision\"",
        "resource.label.\"service_name\"=\"${var.service_name}\"",
        "resource.label.\"location\"=\"${each.key}\""
      ])

      # Bad requests
      bad_service_filter = join(" AND ", [
        "metric.type=\"run.googleapis.com/request_count\"",
        "resource.type=\"cloud_run_revision\"",
        "resource.label.\"service_name\"=\"${var.service_name}\"",
        "metric.label.response_code_class=\"5xx\""
      ])
    }
  }
}

resource "google_monitoring_slo" "success_gclb" {
  count = var.slo.monitor_gclb ? 1 : 0

  service      = google_monitoring_custom_service.this.service_id
  slo_id       = "${var.service_name}-success-gclb-multi-region"
  display_name = "${var.service_name} - Multi-region Success GCLB SLO"
  project      = var.project_id

  goal                = var.slo.success.multi_region_goal
  rolling_period_days = 30

  request_based_sli {
    good_total_ratio {
      # Total requests
      total_service_filter = join(" AND ", [
        "metric.type=\"loadbalancing.googleapis.com/https/request_count\"",
        "resource.type=\"https_lb_rule\"",
        "resource.label.\"backend_name\"=\"${var.service_name}\""
      ])

      # Bad requests
      bad_service_filter = join(" AND ", [
        "metric.type=\"loadbalancing.googleapis.com/https/request_count\"",
        "resource.type=\"https_lb_rule\"",
        "resource.label.\"backend_name\"=\"${var.service_name}\"",
        "metric.label.\"response_code_class\"=\"500\""
      ])
    }
  }
}

# Alert policy for multi-region success SLO burn rate
resource "google_monitoring_alert_policy" "slo_burn_rate_multi_region" {
  count = var.slo.enable_alerting ? 1 : 0

  display_name = "${var.service_name} - Multi-region SLO Burn Rate Alert"
  project      = var.project_id
  combiner     = "OR"

  conditions {
    display_name = "Multi-region SLO burn rate too high"

    condition_threshold {
      filter = <<-EOT
        select_slo_burn_rate("${google_monitoring_slo.success_cr.id}", "60m")
      EOT

      duration        = "0s"
      comparison      = "COMPARISON_GT"
      threshold_value = 10

      aggregations {
        alignment_period   = "60s"
        per_series_aligner = "ALIGN_NEXT_OLDER"
      }
    }
  }

  notification_channels = var.notification_channels

  alert_strategy {
    auto_close = "604800s" # 7 days
  }

  documentation {
    content   = <<-EOT
      The multi-region SLO for ${var.service_name} is burning through its error budget too quickly.

      Current SLO target: ${var.slo.success.multi_region_goal * 100}% success over 30 days
      Monitored regions: us-central1, us-west1, us-east1

      Please investigate the Cloud Run service for errors or performance issues across regions.
    EOT
    mime_type = "text/markdown"
  }
}

# Alert policies for per-region success SLO burn rates
resource "google_monitoring_alert_policy" "slo_burn_rate_per_region" {
  for_each = var.slo.enable_alerting ? toset(var.regions) : []

  display_name = "${var.service_name} - ${each.value} SLO Burn Rate Alert"
  project      = var.project_id
  combiner     = "OR"

  conditions {
    display_name = "${each.value} SLO burn rate too high"

    condition_threshold {
      filter = <<-EOT
        select_slo_burn_rate("${google_monitoring_slo.success_cr_per_region[each.value].id}", "60m")
      EOT

      duration        = "0s"
      comparison      = "COMPARISON_GT"
      threshold_value = 10

      aggregations {
        alignment_period   = "60s"
        per_series_aligner = "ALIGN_NEXT_OLDER"
      }
    }
  }

  notification_channels = var.notification_channels

  alert_strategy {
    auto_close = "604800s" # 7 days
  }

  documentation {
    content   = <<-EOT
      The SLO for ${var.service_name} in ${each.value} is burning through its error budget too quickly.

      Current SLO target: ${var.slo.success.per_region_goal * 100}% success over 30 days

      Please investigate the Cloud Run service in ${each.value} for errors or performance issues.
    EOT
    mime_type = "text/markdown"
  }
}

# Alert policy for success GCLB SLO burn rate
resource "google_monitoring_alert_policy" "slo_burn_rate_gclb" {
  count = var.slo.enable_alerting && var.slo.monitor_gclb ? 1 : 0

  display_name = "${var.service_name} - GCLB SLO Burn Rate Alert"
  project      = var.project_id
  combiner     = "OR"

  conditions {
    display_name = "GCLB SLO burn rate too high"

    condition_threshold {
      filter = <<-EOT
        select_slo_burn_rate("${google_monitoring_slo.success_gclb[0].id}", "60m")
      EOT

      duration        = "0s"
      comparison      = "COMPARISON_GT"
      threshold_value = 10

      aggregations {
        alignment_period   = "60s"
        per_series_aligner = "ALIGN_NEXT_OLDER"
      }
    }
  }

  notification_channels = var.notification_channels

  alert_strategy {
    auto_close = "604800s" # 7 days
  }

  documentation {
    content   = <<-EOT
      The GCLB SLO for ${var.service_name} is burning through its error budget too quickly.

      Current SLO target: ${var.slo.success.multi_region_goal * 100}% success over 30 days

      Please investigate the Cloud Run service for errors or performance issues across regions.
    EOT
    mime_type = "text/markdown"
  }
}
