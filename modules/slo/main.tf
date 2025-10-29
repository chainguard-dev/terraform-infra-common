locals {
  rolling_periods = {
    "07" = 7
    "30" = 30
  }
  region_rolling_period_map = {
    for pair in setproduct(var.regions, keys(local.rolling_periods)) :
    "${pair[0]}-${pair[1]}" => {
      region              = pair[0]
      rolling_period_key  = pair[1]
      rolling_period_days = local.rolling_periods[pair[1]]
    }
  }
}

resource "google_monitoring_custom_service" "this" {
  service_id = "${lower(var.service_type)}-${lower(var.service_name)}"
  project    = var.project_id
}

resource "google_monitoring_slo" "success_cr" {
  for_each = local.rolling_periods

  service      = google_monitoring_custom_service.this.service_id
  slo_id       = "${var.service_name}-success-multi-region-${each.key}d"
  display_name = "${var.service_name} - Multi-region Success ${each.key}d Cloud Run SLO"
  project      = var.project_id

  goal                = var.slo.success.multi_region_goal
  rolling_period_days = each.value

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
  for_each = local.region_rolling_period_map

  service      = google_monitoring_custom_service.this.service_id
  slo_id       = "${var.service_name}-success-${each.value.region}-${each.value.rolling_period_key}d"
  display_name = "${var.service_name} - ${each.key} Success ${each.value.rolling_period_key}d Cloud Run SLO"
  project      = var.project_id

  goal                = var.slo.success.per_region_goal
  rolling_period_days = each.value.rolling_period_days

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
  for_each = var.slo.monitor_gclb ? local.rolling_periods : {}

  service      = google_monitoring_custom_service.this.service_id
  slo_id       = "${var.service_name}-success-gclb-multi-region-${each.key}d"
  display_name = "${var.service_name} - Multi-region Success ${each.key}d GCLB SLO"
  project      = var.project_id

  goal                = var.slo.success.multi_region_goal
  rolling_period_days = each.value

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
  for_each = var.slo.monitor_gclb ? local.rolling_periods : {}

  display_name = "${var.service_name} - Multi-region ${each.key}d SLO Burn Rate Alert"
  project      = var.project_id
  combiner     = "OR"

  conditions {
    display_name = "Multi-region ${each.key}d SLO burn rate too high"

    condition_threshold {
      filter = <<-EOT
        select_slo_burn_rate("${google_monitoring_slo.success_cr[each.key].id}", "60m")
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

      Current SLO target: ${var.slo.success.multi_region_goal * 100}% success over ${each.key} days
      Monitored regions: us-central1, us-west1, us-east1

      Please investigate the Cloud Run service for errors or performance issues across regions.
    EOT
    mime_type = "text/markdown"
  }
}

# Alert policies for per-region success SLO burn rates
resource "google_monitoring_alert_policy" "slo_burn_rate_per_region" {
  for_each = var.slo.enable_alerting ? local.region_rolling_period_map : {}

  display_name = "${var.service_name} - ${each.value.region} ${each.value.rolling_period_key}d SLO Burn Rate Alert"
  project      = var.project_id
  combiner     = "OR"

  conditions {
    display_name = "${var.service_name} ${each.value.region} ${each.value.rolling_period_key}d SLO burn rate too high"

    condition_threshold {
      filter = <<-EOT
        select_slo_burn_rate("${google_monitoring_slo.success_cr_per_region[each.key].id}", "60m")
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
      The SLO for ${var.service_name} in ${each.value.region} is burning through its error budget too quickly.

      Current SLO target: ${var.slo.success.per_region_goal * 100}% success over ${each.value.rolling_period_key} days

      Please investigate the Cloud Run service in ${each.value.region} for errors or performance issues.
    EOT
    mime_type = "text/markdown"
  }
}

# Alert policy for success GCLB SLO burn rate
resource "google_monitoring_alert_policy" "slo_burn_rate_gclb" {
  for_each = var.slo.enable_alerting && var.slo.monitor_gclb ? local.rolling_periods : {}

  display_name = "${var.service_name} - GCLB ${each.key}d SLO Burn Rate Alert"
  project      = var.project_id
  combiner     = "OR"

  conditions {
    display_name = "GCLB ${each.key}d SLO burn rate too high"

    condition_threshold {
      filter = <<-EOT
        select_slo_burn_rate("${google_monitoring_slo.success_gclb[each.key].id}", "60m")
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

      Current SLO target: ${var.slo.success.multi_region_goal * 100}% success over ${each.key} days

      Please investigate the Cloud Run service for errors or performance issues across regions.
    EOT
    mime_type = "text/markdown"
  }
}
