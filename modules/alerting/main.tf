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
        "job_name"      = "EXTRACT(resource.labels.job_name)"
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
        resource.type="cloud_run_revision" OR resource.type="cloud_run_job"
        severity=ERROR
        textPayload=~"panic: .*"
      EOT

      label_extractors = {
        "revision_name" = "EXTRACT(resource.labels.revision_name)"
        "job_name"      = "EXTRACT(resource.labels.job_name)"
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
        resource.type="cloud_run_revision" OR resource.type="cloud_run_job"
        jsonPayload.stacktrace:"runtime.gopanic"
      EOT

      label_extractors = {
        "revision_name" = "EXTRACT(resource.labels.revision_name)"
        "job_name"      = "EXTRACT(resource.labels.job_name)"
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
        resource.type="cloud_run_revision" OR resource.type="cloud_run_job"
        textPayload:"fatal error: "
      EOT

      label_extractors = {
        "revision_name" = "EXTRACT(resource.labels.revision_name)"
        "job_name"      = "EXTRACT(resource.labels.job_name)"
        "location"      = "EXTRACT(resource.labels.location)"
      }
    }
  }

  notification_channels = length(var.notification_channels) != 0 ? var.notification_channels : local.slack

  enabled = "true"
  project = var.project_id
}

resource "google_monitoring_alert_policy" "service_failure_rate_non_eventing" {
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
           (rate(http_request_status_total{service_name!~"${join("|", var.failure_rate_exclude_services)}", code=~"5..", ce_type!~"dev.chainguard.*"}[1m]))
         /
         sum by (service_name)
           (rate(http_request_status_total{service_name!~"${join("|", var.failure_rate_exclude_services)}", ce_type!~"dev.chainguard.*"}[1m]))
        ) > ${var.failure_rate_ratio_threshold}
        and
        sum by (service_name)
          (rate(http_request_status_total{service_name!~"${join("|", var.failure_rate_exclude_services)}", ce_type!~"dev.chainguard.*"}[5m]))
        > 0.0001
      EOT
    }

    display_name = "cloudrun service 5xx failure rate above ${var.failure_rate_ratio_threshold}"
  }

  display_name = "cloudrun service 5xx failure rate above ${var.failure_rate_ratio_threshold}"

  documentation {
    // variables reference: https://cloud.google.com/monitoring/alerts/doc-variables#doc-vars
    subject = "Cloud Run service $${metric_or_resource.labels.service_name} had 5xx error rate above ${var.failure_rate_ratio_threshold} for ${var.failure_rate_duration}s"

    content = <<-EOT
    Please consult the playbook entry [here](https://wiki.inky.wtf/docs/teams/engineering/enforce/playbooks/5xx/) for troubleshooting information.
    EOT
  }

  notification_channels = length(var.notification_channels) != 0 ? var.notification_channels : local.oncall

  enabled = "true"
  project = var.project_id
}

resource "google_monitoring_alert_policy" "service_failure_rate_eventing" {
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
           (rate(http_request_status_total{service_name!~"${join("|", var.failure_rate_exclude_services)}", code=~"5..", ce_type=~"dev.chainguard.*"}[1m]))
         /
         sum by (service_name)
           (rate(http_request_status_total{service_name!~"${join("|", var.failure_rate_exclude_services)}", ce_type=~"dev.chainguard.*"}[1m]))
        ) > ${var.failure_rate_ratio_threshold}
        and
        sum by (service_name)
          (rate(http_request_status_total{service_name!~"${join("|", var.failure_rate_exclude_services)}", ce_type=~"dev.chainguard.*"}[5m]))
        > 0.0001
      EOT
    }

    display_name = "eventing services 5xx failure rate above ${var.failure_rate_ratio_threshold}"
  }

  display_name = "eventing services 5xx failure rate above ${var.failure_rate_ratio_threshold}"

  documentation {
    // variables reference: https://cloud.google.com/monitoring/alerts/doc-variables#doc-vars
    subject = "Eventing service $${metric_or_resource.labels.service_name} had 5xx error rate above ${var.failure_rate_ratio_threshold} for ${var.failure_rate_duration}s"

    content = <<-EOT
    Please consult the playbook entry [here](https://wiki.inky.wtf/docs/teams/engineering/enforce/playbooks/5xx/) for troubleshooting information.
    EOT
  }

  notification_channels = length(var.notification_channels) != 0 ? var.notification_channels : local.slack

  enabled = "true"
  project = var.project_id
}

resource "google_logging_metric" "cloud-run-scaling-failure" {
  name   = "cloud_run_scaling_failure"
  filter = <<EOT
        resource.type="cloud_run_revision"
        log_name="projects/${var.project_id}/logs/run.googleapis.com%2Frequests"
        severity=ERROR
        textPayload:"The request was aborted because there was no available instance."
      EOT
  metric_descriptor {
    metric_kind = "DELTA"
    value_type  = "INT64"
    labels {
      key         = "location"
      value_type  = "STRING"
      description = "location of service."
    }
    labels {
      key         = "service_name"
      value_type  = "STRING"
      description = "name of service."
    }
  }

  label_extractors = {
    "location"     = "EXTRACT(resource.labels.location)"
    "service_name" = "EXTRACT(resource.labels.service_name)"
  }
}

resource "google_monitoring_alert_policy" "cloud-run-scaling-failure" {
  # In the absence of data, incident will auto-close after an daily
  alert_strategy {
    auto_close = "86400s"

    notification_rate_limit {
      period = "86400s" // re-alert daily if condition still valid.
    }
  }

  display_name = "Cloud Run scaling issue"
  combiner     = "OR"

  conditions {
    display_name = "Cloud Run scaling issue"

    condition_matched_log {
      filter = <<EOT
        resource.type="cloud_run_revision"
        log_name="projects/${var.project_id}/logs/run.googleapis.com%2Frequests"
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

resource "google_logging_metric" "cloud-run-failed-req" {
  name   = "cloud_run_failed_req"
  filter = <<EOT
        resource.type="cloud_run_revision"
        log_name="projects/${var.project_id}/logs/run.googleapis.com%2Frequests"
        severity=ERROR
        textPayload:"The request failed because either the HTTP response was malformed or connection to the instance had an error."
      EOT
  metric_descriptor {
    metric_kind = "DELTA"
    value_type  = "INT64"
    labels {
      key         = "location"
      value_type  = "STRING"
      description = "location of service."
    }
    labels {
      key         = "service_name"
      value_type  = "STRING"
      description = "name of service."
    }
  }

  label_extractors = {
    "location"     = "EXTRACT(resource.labels.location)"
    "service_name" = "EXTRACT(resource.labels.service_name)"
  }
}

resource "google_monitoring_alert_policy" "cloud-run-failed-req" {
  # In the absence of data, incident will auto-close after an daily
  alert_strategy {
    auto_close = "86400s"

    notification_rate_limit {
      period = "86400s" // re-alert daily if condition still valid.
    }
  }

  display_name = "Cloud Run failed request"
  combiner     = "OR"

  conditions {
    display_name = "Cloud Run failed request"

    condition_matched_log {
      filter = <<EOT
        resource.type="cloud_run_revision"
        log_name="projects/${var.project_id}/logs/run.googleapis.com%2Frequests"
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

resource "google_monitoring_alert_policy" "pubsub_dead_letter_queue_messages" {
  alert_strategy {
    auto_close = "3600s" // 1 hour
  }

  combiner = "OR"

  conditions {
    condition_threshold {
      aggregations {
        alignment_period   = "600s"
        per_series_aligner = "ALIGN_MAX"
      }

      comparison = "COMPARISON_GT"
      duration   = "0s"
      filter     = <<EOT
        metric.type="pubsub.googleapis.com/topic/send_request_count"
        resource.type="pubsub_topic"
        metadata.system_labels."name"=monitoring.regex.full_match(".*-dlq-.*")
        ${var.dlq_filter}
      EOT

      trigger {
        count = "1"
      }

      // TODO: make configurable later
      threshold_value = 1
    }

    display_name = "Dead-letter queue messages above 1"
  }
  display_name = "Dead-letter queue messages above 1"

  notification_channels = length(var.notification_channels) != 0 ? var.notification_channels : local.slack

  enabled = "true"
  project = var.project_id
}
