locals {
  region_filter = "one_of(${join(", ", formatlist("%q", var.regions))})"
}

resource "google_monitoring_service" "this" {
  service_id = "${lower(var.service_type)}-${lower(var.name)}"
  project    = var.project_id

  basic_service {
    service_type = var.service_type
    service_labels = {
      service_name = var.service_name
    }
  }
}

# SLO with availability SLI across all regions
resource "google_monitoring_slo" "availability" {
  service      = google_monitoring_service.this.service_id
  slo_id       = "${var.service_name}-availability-multi-region"
  display_name = "${var.service_name} - Multi-region Availability SLO"
  project      = var.project_id

  goal                = var.slo.availability.multi_region_goal
  rolling_period_days = 30

  basic_sli {
    availability {}
  }
}

resource "google_monitoring_slo" "availability_per_region" {
  for_each = toset(var.regions)

  service      = google_monitoring_service.this.service_id
  slo_id       = "${var.service_name}-availability-${each.value}"
  display_name = "${var.service_name} - ${each.value} Availability SLO"
  project      = var.project_id

  goal                = var.slo.availability.per_region_goal
  rolling_period_days = 30

  basic_sli {
    availability {}
  }
}

# Alert policy for multi-region SLO burn rate
resource "google_monitoring_alert_policy" "slo_burn_rate_multi_region" {
  count = var.slo.enable_alerting ? 1 : 0

  display_name = "${var.service_name} - Multi-region SLO Burn Rate Alert"
  project      = var.project_id
  combiner     = "OR"

  conditions {
    display_name = "Multi-region SLO burn rate too high"

    condition_threshold {
      filter = <<-EOT
        select_slo_burn_rate("${google_monitoring_slo.availability.id}", 3600)
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
      
      Current SLO target: ${var.availability.multi_region_goal * 100}% availability over 30 days
      Monitored regions: us-central1, us-west1, us-east1
      
      Please investigate the Cloud Run service for errors or performance issues across regions.
    EOT
    mime_type = "text/markdown"
  }
}

# Alert policies for per-region SLO burn rates
resource "google_monitoring_alert_policy" "slo_burn_rate_per_region" {
  count = var.slo.enable_alerting ? 1 : 0

  for_each = toset(var.regions)

  display_name = "${var.service_name} - ${each.value} SLO Burn Rate Alert"
  project      = var.project_id
  combiner     = "OR"

  conditions {
    display_name = "${each.value} SLO burn rate too high"

    condition_threshold {
      filter = <<-EOT
        select_slo_burn_rate("${google_monitoring_slo.availability_per_region[each.value].id}", 3600)
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
      
      Current SLO target: ${var.availability.per_region_goal * 100}% availability over 30 days
      
      Please investigate the Cloud Run service in ${each.value} for errors or performance issues.
    EOT
    mime_type = "text/markdown"
  }
}
