locals {
  oncall = [
    var.notification_channel_pagerduty,
    var.notification_channel_email
  ]

  slack = [
    var.notification_channel_slack,
    var.notification_channel_email
  ]
}

// Create an alert policy to notify if the service is struggling to rollout.
resource "google_monitoring_alert_policy" "bad-rollout" {
  # In the absence of data, incident will auto-close after an hour
  alert_strategy {
    auto_close = "3600s"

    notification_rate_limit {
      period = "3600s" // re-alert hourly if condition still valid.
    }
  }

  display_name = "Failed Revision Rollout"
  combiner     = "OR"

  conditions {
    display_name = "Failed Revision Rollout"

    condition_matched_log {
      filter = <<EOT
        resource.type="cloud_run_revision"
        severity=ERROR
        protoPayload.status.message:"Ready condition status changed to False"
        protoPayload.response.kind="Revision"
      EOT

      label_extractors = {
        "revision_name" = "EXTRACT(resource.labels.revision_name)"
        "location"      = "EXTRACT(resource.labels.location)"
      }
    }
  }

  notification_channels = length(var.notification_channels) != 0 ? var.notification_channels : local.oncall

  enabled = "true"
  project = var.project_id
}

resource "google_monitoring_alert_policy" "oom" {
  # In the absence of data, incident will auto-close after an hour
  alert_strategy {
    auto_close = "3600s"

    notification_rate_limit {
      period = "3600s" // re-alert once an hour if condition still valid.
    }
  }

  display_name = "OOM Alert"
  combiner     = "OR"

  conditions {
    display_name = "OOM Alert"

    condition_matched_log {
      filter = <<EOT
        logName: "/logs/run.googleapis.com%2Fvarlog%2Fsystem"
        severity=ERROR
        textPayload:"Consider increasing the memory limit"
        ${var.oom_filter}
      EOT

      label_extractors = {
        "revision_name" = "EXTRACT(resource.labels.revision_name)"
        "location"      = "EXTRACT(resource.labels.location)"
      }
    }
  }

  enabled = true

  notification_channels = length(var.notification_channels) != 0 ? var.notification_channels : local.slack
}

resource "google_monitoring_alert_policy" "panic" {
  # In the absence of data, incident will auto-close after an hour
  alert_strategy {
    auto_close = "3600s"

    notification_rate_limit {
      period = "3600s" // re-alert hourly if condition still valid.
    }
  }

  display_name = "Panic log entry"
  combiner     = "OR"

  conditions {
    display_name = "Panic log entry"

    condition_matched_log {
      filter = <<EOT
        resource.type="cloud_run_revision"
        severity=ERROR
        textPayload=~"panic: .*"
      EOT

      label_extractors = {
        "revision_name" = "EXTRACT(resource.labels.revision_name)"
        "location"      = "EXTRACT(resource.labels.location)"
      }
    }
  }

  notification_channels = length(var.notification_channels) != 0 ? var.notification_channels : local.slack

  enabled = "true"
  project = var.project_id
}

resource "google_monitoring_alert_policy" "panic-stacktrace" {
  # In the absence of data, incident will auto-close after an hour
  alert_strategy {
    auto_close = "3600s"

    notification_rate_limit {
      period = "3600s" // re-alert hourly if condition still valid.
    }
  }

  display_name = "Panic stacktrace log entry"
  combiner     = "OR"

  conditions {
    display_name = "Panic stacktrace log entry"

    condition_matched_log {
      filter = <<EOT
        resource.type="cloud_run_revision"
        jsonPayload.stacktrace:"runtime.gopanic"
      EOT

      label_extractors = {
        "revision_name" = "EXTRACT(resource.labels.revision_name)"
        "location"      = "EXTRACT(resource.labels.location)"
      }
    }
  }

  notification_channels = length(var.notification_channels) != 0 ? var.notification_channels : local.slack

  enabled = "true"
  project = var.project_id
}

resource "google_monitoring_alert_policy" "fatal" {
  # In the absence of data, incident will auto-close after an hour
  alert_strategy {
    auto_close = "3600s"

    notification_rate_limit {
      period = "3600s" // re-alert hourly if condition still valid.
    }
  }

  display_name = "Fatal log entry"
  combiner     = "OR"

  conditions {
    display_name = "Fatal log entry"

    condition_matched_log {
      filter = <<EOT
        resource.type="cloud_run_revision"
        textPayload:"fatal error: "
      EOT

      label_extractors = {
        "revision_name" = "EXTRACT(resource.labels.revision_name)"
        "location"      = "EXTRACT(resource.labels.location)"
      }
    }
  }

  notification_channels = length(var.notification_channels) != 0 ? var.notification_channels : local.slack

  enabled = "true"
  project = var.project_id
}

resource "google_monitoring_alert_policy" "service_failure_rate" {
  # In the absence of data, incident will auto-close after an hour
  alert_strategy {
    auto_close = "3600s"
  }

  combiner = "OR"

  conditions {
    condition_prometheus_query_language {
      duration            = "${var.failure_rate_duration}s"
      evaluation_interval = "60s"

      // Using custom prometheus metric to avoid counting failed health check 5xx, should be a separate alert
      // First part of the query calculates the error rate (5xx / all) and the rate should be greater than var.failure_rate_ratio_threshold
      // Second part ensures services has non-zero traffic over last 5 min.
      query = <<EOT
        (sum by (service_name)
           (rate(http_request_status_total{code=~"5.."}[1m]))
         /
         sum by (service_name)
           (rate(http_request_status_total{}[1m]))
        ) > ${var.failure_rate_ratio_threshold}
        and
        sum by (service_name)
          (rate(http_request_status_total{}[5m]))
        > 0.0001
      EOT
    }

    display_name = "5xx failure rate above ${var.failure_rate_ratio_threshold}"
  }

  display_name = "5xx failure rate above ${var.failure_rate_ratio_threshold}"

  documentation {
    // variables reference: https://cloud.google.com/monitoring/alerts/doc-variables#doc-vars
    subject = "Cloud Run service $${resource.label.service_name} had 5xx error rate above ${var.failure_rate_ratio_threshold} for ${var.failure_rate_duration}s"

    content = <<-EOT
    Please consult the playbook entry [here](https://wiki.inky.wtf/docs/teams/engineering/enforce/playbooks/5xx/) for troubleshooting information.
    EOT
  }

  notification_channels = length(var.notification_channels) != 0 ? var.notification_channels : local.oncall

  enabled = "true"
  project = var.project_id
}

resource "google_monitoring_alert_policy" "cloud-run-scaling-failure" {
  # In the absence of data, incident will auto-close after an hour
  alert_strategy {
    auto_close = "3600s"

    notification_rate_limit {
      period = "3600s" // re-alert hourly if condition still valid.
    }
  }

  display_name = "Cloud Run scaling issue"
  combiner     = "OR"

  conditions {
    display_name = "Cloud Run scaling issue"

    condition_matched_log {
      filter = <<EOT
        resource.type="cloud_run_revision"
        log_name="projects/prod-enforce-fabc/logs/run.googleapis.com%2Frequests"
        severity=ERROR
        textPayload:"The request was aborted because there was no available instance."
        ${var.scaling_issue_filter}
      EOT

      label_extractors = {
        "revision_name" = "EXTRACT(resource.labels.revision_name)"
        "location"      = "EXTRACT(resource.labels.location)"
      }
    }
  }

  notification_channels = length(var.notification_channels) != 0 ? var.notification_channels : local.slack

  enabled = "true"
  project = var.project_id
}

resource "google_monitoring_alert_policy" "cloud-run-failed-req" {
  # In the absence of data, incident will auto-close after an hour
  alert_strategy {
    auto_close = "3600s"

    notification_rate_limit {
      period = "3600s" // re-alert hourly if condition still valid.
    }
  }

  display_name = "Cloud Run failed request"
  combiner     = "OR"

  conditions {
    display_name = "Cloud Run failed request"

    condition_matched_log {
      filter = <<EOT
        resource.type="cloud_run_revision"
        log_name="projects/prod-enforce-fabc/logs/run.googleapis.com%2Frequests"
        severity=ERROR
        textPayload:"The request failed because either the HTTP response was malformed or connection to the instance had an error."
        ${var.failed_req_filter}
      EOT

      label_extractors = {
        "revision_name" = "EXTRACT(resource.labels.revision_name)"
        "location"      = "EXTRACT(resource.labels.location)"
      }
    }
  }

  notification_channels = length(var.notification_channels) != 0 ? var.notification_channels : local.slack

  enabled = "true"
  project = var.project_id
}
