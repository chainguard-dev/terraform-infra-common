/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Personalized Service Health writes an EventLog entry to Cloud Logging each
// time a Google Cloud incident relevant to the project is created or updated.
// A log-match alert policy turns those entries into notifications.
// https://docs.cloud.google.com/service-health/docs/configure-alerts-terraform
locals {
  notification_channels = distinct(concat(
    var.notification_channels_slack,
    var.notification_channels_email,
    var.notification_channels_pubsub,
    var.notification_channels,
  ))

  products_filter  = length(var.products) == 0 ? "" : "AND (${join(" OR ", [for p in var.products : "jsonPayload.impactedProducts:${jsonencode(p)}"])})"
  locations_filter = length(var.locations) == 0 ? "" : "AND (${join(" OR ", [for l in var.locations : "jsonPayload.impactedLocations:${jsonencode(l)}"])})"

  filter = <<EOT
resource.type="servicehealth.googleapis.com/Event"
AND jsonPayload.@type="type.googleapis.com/google.cloud.servicehealth.logging.v1.EventLog"
AND jsonPayload.category="INCIDENT"
${local.products_filter}
${local.locations_filter}
EOT

  subject_prefix = var.name == "" ? "" : "[${var.name}] "
}

resource "google_project_service" "servicehealth" {
  count = var.enable_api ? 1 : 0

  project            = var.project_id
  service            = "servicehealth.googleapis.com"
  disable_on_destroy = false
}

resource "google_monitoring_alert_policy" "incident" {
  project      = var.project_id
  display_name = "${local.subject_prefix}Google Cloud Service Health incident"
  combiner     = "OR"
  severity     = var.severity

  conditions {
    display_name = "Service Health incident created or updated"

    condition_matched_log {
      filter = local.filter

      label_extractors = {
        title             = "EXTRACT(jsonPayload.title)"
        state             = "EXTRACT(jsonPayload.state)"
        detailedState     = "EXTRACT(jsonPayload.detailedState)"
        relevance         = "EXTRACT(jsonPayload.relevance)"
        description       = "EXTRACT(jsonPayload.description)"
        impactedProducts  = "EXTRACT(jsonPayload.impactedProducts)"
        impactedLocations = "EXTRACT(jsonPayload.impactedLocations)"
        startTime         = "EXTRACT(jsonPayload.startTime)"
        endTime           = "EXTRACT(jsonPayload.endTime)"
      }
    }
  }

  documentation {
    // variables reference: https://cloud.google.com/monitoring/alerts/doc-variables#doc-vars
    subject   = "${local.subject_prefix}GCP incident ($${log.extracted_label.state}): $${log.extracted_label.title}"
    mime_type = "text/markdown"
    content   = <<EOT
### $${log.extracted_label.title}

**State:** $${log.extracted_label.state} ($${log.extracted_label.detailedState})
**Relevance:** $${log.extracted_label.relevance}
**Products:** $${log.extracted_label.impactedProducts}
**Locations:** $${log.extracted_label.impactedLocations}
**Started:** $${log.extracted_label.startTime}
**Ended:** $${log.extracted_label.endTime}

$${log.extracted_label.description}

---
**What this means:** Google reports an incident affecting a product or location that project ${var.project_id} uses. It may or may not be degrading our services.

**What to do:**
1. Open the Service Health dashboard for impact details and Google's next update: https://console.cloud.google.com/servicehealth/incidents?project=${var.project_id}
2. Check dashboards and error rates for our services that run on the affected products and locations.
3. If our services are degraded, declare an incident and link the GCP incident. Otherwise, watch for the CLOSED update.
EOT

    links {
      display_name = "Service Health dashboard"
      url          = "https://console.cloud.google.com/servicehealth/incidents?project=${var.project_id}"
    }
  }

  alert_strategy {
    auto_close = var.auto_close

    notification_rate_limit {
      period = var.notification_rate_limit
    }
  }

  notification_channels = local.notification_channels

  lifecycle {
    precondition {
      condition     = length(local.notification_channels) > 0
      error_message = "At least one notification channel must be set (notification_channels_slack, notification_channels_email, notification_channels_pubsub, or notification_channels)."
    }
  }

  depends_on = [google_project_service.servicehealth]
}
