locals {
  oncall = [
    var.notification_channel_pagerduty,
    var.notification_channel_email,
    var.notification_channel_pubsub
  ]

  slack = [
    var.notification_channel_slack,
    var.notification_channel_email,
    var.notification_channel_pubsub
  ]
}

locals {
  squad_log_filter = var.squad == "" ? "" : "labels.squad=\"${var.squad}\""
  name             = var.squad == "" ? "global" : var.squad
}

locals {
  bad_rollout_filter = <<EOT
resource.type="cloud_run_revision"
severity=ERROR
protoPayload.status.message:"Ready condition status changed to False"
protoPayload.response.kind="Revision"
${local.squad_log_filter}
EOT
}

// Create an alert policy to notify if the service is struggling to rollout.
resource "google_monitoring_alert_policy" "bad-rollout" {
  count = var.global_only_alerts ? 0 : 1

  # In the absence of data, incident will auto-close after an hour
  alert_strategy {
    auto_close = "3600s"

    notification_rate_limit {
      period = "3600s" // re-alert hourly if condition still valid.
    }
  }

  display_name = "Failed Revision Rollout ${local.name}"
  combiner     = "OR"

  documentation {
    content = "$${metric_or_resource.labels.service_name} has failed to rollout a revision."
    links {
      display_name = "Logs Explorer"
      url          = "https://console.cloud.google.com/logs/query;query=${urlencode(local.bad_rollout_filter)}?project=${var.project_id}"
    }
  }

  conditions {
    display_name = "Failed Revision Rollout ${local.name}"

    condition_matched_log {
      filter = local.bad_rollout_filter

      label_extractors = {
        "service_name"  = "EXTRACT(resource.labels.service_name)"
        "revision_name" = "EXTRACT(resource.labels.revision_name)"
        "location"      = "EXTRACT(resource.labels.location)"
      }
    }
  }

  notification_channels = length(var.notification_channels) != 0 ? var.notification_channels : local.slack

  enabled = "true"
  project = var.project_id
}

locals {
  oom_filter = <<EOF
logName: "/logs/run.googleapis.com%2Fvarlog%2Fsystem"
severity=ERROR
textPayload:"Consider increasing the memory limit"
${var.oom_filter}
${local.squad_log_filter}
EOF
}

resource "google_monitoring_alert_policy" "oom" {
  count = var.global_only_alerts ? 0 : 1

  # In the absence of data, incident will auto-close after an hour
  alert_strategy {
    auto_close = "3600s"

    notification_rate_limit {
      period = "3600s" // re-alert once an hour if condition still valid.
    }
  }

  display_name = "OOM Alert ${local.name}"
  combiner     = "OR"

  documentation {
    content = "$${metric_or_resource.labels.service_name}$${metric_or_resource.labels.job_name} has logged an OOM."
    links {
      display_name = "Logs Explorer"
      url          = "https://console.cloud.google.com/logs/query;query=${urlencode(local.oom_filter)}?project=${var.project_id}"
    }
  }

  conditions {
    display_name = "OOM Alert ${local.name}"

    condition_matched_log {
      filter = local.oom_filter

      label_extractors = {
        "service_name"  = "EXTRACT(resource.labels.service_name)"
        "revision_name" = "EXTRACT(resource.labels.revision_name)"
        "job_name"      = "EXTRACT(resource.labels.job_name)"
        "location"      = "EXTRACT(resource.labels.location)"
      }
    }
  }

  enabled = true

  notification_channels = length(var.notification_channels) != 0 ? var.notification_channels : local.slack
}

locals {
  signal_filter = <<EOT
log_name="projects/${var.project_id}/logs/run.googleapis.com%2Fvarlog%2Fsystem"
severity=WARNING
textPayload=~"^Container terminated on signal [^01]+\.$"
${var.signal_filter}
-resource.labels.service_name:"-ing-vuln"
${local.squad_log_filter}
EOT
}

resource "google_monitoring_alert_policy" "signal" {
  count = var.global_only_alerts ? 0 : 1

  # In the absence of data, incident will auto-close after an hour
  alert_strategy {
    auto_close = "3600s"

    notification_rate_limit {
      period = "3600s" // re-alert once an hour if condition still valid.
    }
  }

  display_name = "Signal Alert ${local.name}"
  combiner     = "OR"

  documentation {
    content = "$${metric_or_resource.labels.service_name}$${metric_or_resource.labels.job_name} has logged a termination signal."
    links {
      display_name = "Logs Explorer"
      url          = "https://console.cloud.google.com/logs/query;query=${urlencode(local.oom_filter)}?project=${var.project_id}"
    }
  }

  conditions {
    display_name = "Signal Alert ${local.name}"

    condition_matched_log {
      filter = local.signal_filter

      label_extractors = {
        "service_name"  = "EXTRACT(resource.labels.service_name)"
        "revision_name" = "EXTRACT(resource.labels.revision_name)"
        "job_name"      = "EXTRACT(resource.labels.job_name)"
        "location"      = "EXTRACT(resource.labels.location)"
        "signal"        = "REGEXP_EXTRACT(textPayload, \"^Container terminated on signal ([^01]+)\\.$\")"
      }
    }
  }

  enabled = true

  notification_channels = length(var.notification_channels) != 0 ? var.notification_channels : local.slack
}

locals {
  panic_filter = <<EOF
resource.type="cloud_run_revision" OR resource.type="cloud_run_job"
severity=ERROR
textPayload=~"panic: .*"
${var.panic_filter}
${local.squad_log_filter}
EOF
}

resource "google_monitoring_alert_policy" "panic" {
  count = var.global_only_alerts ? 0 : 1

  # In the absence of data, incident will auto-close after an hour
  alert_strategy {
    auto_close = "3600s"

    notification_rate_limit {
      period = "3600s" // re-alert hourly if condition still valid.
    }
  }

  display_name = "Panic log entry ${local.name}"
  combiner     = "OR"

  documentation {
    content = "$${metric_or_resource.labels.service_name}$${metric_or_resource.labels.job_name} has logged a panic."
    links {
      display_name = "Logs Explorer"
      url          = "https://console.cloud.google.com/logs/query;query=${urlencode(local.panic_filter)}?project=${var.project_id}"
    }
  }

  conditions {
    display_name = "Panic log entry ${local.name}"

    condition_matched_log {
      filter = local.panic_filter

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

locals {
  panic_stacktrace_filter = <<EOF
resource.type="cloud_run_revision" OR resource.type="cloud_run_job"
jsonPayload.stacktrace:"runtime.gopanic"
${local.squad_log_filter}
EOF
}

resource "google_monitoring_alert_policy" "panic-stacktrace" {
  count = var.global_only_alerts ? 0 : 1

  # In the absence of data, incident will auto-close after an hour
  alert_strategy {
    auto_close = "3600s"

    notification_rate_limit {
      period = "3600s" // re-alert hourly if condition still valid.
    }
  }

  display_name = "Panic stacktrace log entry ${local.name}"
  combiner     = "OR"

  documentation {
    content = "$${metric_or_resource.labels.service_name}$${metric_or_resource.labels.job_name} has logged a panic stacktrace."
    links {
      display_name = "Logs Explorer"
      url          = "https://console.cloud.google.com/logs/query;query=${urlencode(local.panic_stacktrace_filter)}&project=${var.project_id}"
    }
  }

  conditions {
    display_name = "Panic stacktrace log entry ${local.name}"

    condition_matched_log {
      filter = local.panic_stacktrace_filter

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

locals {
  fatal_filter = <<EOF
resource.type="cloud_run_revision" OR resource.type="cloud_run_job"
textPayload:"fatal error: "
${local.squad_log_filter}
EOF
}

resource "google_monitoring_alert_policy" "fatal" {
  count = var.global_only_alerts ? 0 : 1

  # In the absence of data, incident will auto-close after an hour
  alert_strategy {
    auto_close = "3600s"

    notification_rate_limit {
      period = "3600s" // re-alert hourly if condition still valid.
    }
  }

  display_name = "Fatal log entry ${local.name}"
  combiner     = "OR"

  documentation {
    content = "$${metric_or_resource.labels.service_name}$${metric_or_resource.labels.job_name} has logged a fatal error."
    links {
      display_name = "Logs Explorer"
      url          = "https://console.cloud.google.com/logs/query;query=${urlencode(local.panic_stacktrace_filter)}&project=${var.project_id}"
    }
  }

  conditions {
    display_name = "Fatal log entry ${local.name}"

    condition_matched_log {
      filter = local.fatal_filter

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

locals {
  promql_squad_filter = var.squad == "" ? "" : ", team=\"${var.squad}\""
}

resource "google_monitoring_alert_policy" "service_failure_rate_non_eventing" {
  count = var.global_only_alerts ? 0 : 1

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
           (rate(http_request_status_total{service_name!~"${join("|", var.failure_rate_exclude_services)}", code=~"5..", ce_type!~"dev.chainguard.*"${local.promql_squad_filter}}[1m]))
         /
         sum by (service_name)
           (rate(http_request_status_total{service_name!~"${join("|", var.failure_rate_exclude_services)}", ce_type!~"dev.chainguard.*"${local.promql_squad_filter}}[1m]))
        ) > ${var.failure_rate_ratio_threshold}
        and
        sum by (service_name)
          (rate(http_request_status_total{service_name!~"${join("|", var.failure_rate_exclude_services)}", ce_type!~"dev.chainguard.*"${local.promql_squad_filter}}[5m]))
        > 0.0001
      EOT
    }

    display_name = "cloudrun service 5xx failure rate above ${var.failure_rate_ratio_threshold} ${local.name}"
  }

  display_name = "cloudrun service 5xx failure rate above ${var.failure_rate_ratio_threshold} ${local.name}"

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
  count = var.global_only_alerts ? 0 : 1

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
           (rate(http_request_status_total{service_name!~"${join("|", var.failure_rate_exclude_services)}", code=~"5..", ce_type=~"dev.chainguard.*"${local.promql_squad_filter}}[1m]))
         /
         sum by (service_name)
           (rate(http_request_status_total{service_name!~"${join("|", var.failure_rate_exclude_services)}", ce_type=~"dev.chainguard.*"${local.promql_squad_filter}}[1m]))
        ) > ${var.failure_rate_ratio_threshold}
        and
        sum by (service_name)
          (rate(http_request_status_total{service_name!~"${join("|", var.failure_rate_exclude_services)}", ce_type=~"dev.chainguard.*"${local.promql_squad_filter}}[5m]))
        > 0.0001
      EOT
    }

    display_name = "eventing services 5xx failure rate above ${var.failure_rate_ratio_threshold} ${local.name}"
  }

  display_name = "eventing services 5xx failure rate above ${var.failure_rate_ratio_threshold} ${local.name}"

  documentation {
    // variables reference: https://cloud.google.com/monitoring/alerts/doc-variables#doc-vars
    subject = "Eventing service $${metric_or_resource.labels.service_name} had 5xx error rate above ${var.failure_rate_ratio_threshold} for ${var.failure_rate_duration}s"

    content = <<-EOT
    Please consult the playbook entry [here](https://wiki.inky.wtf/docs/teams/engineering/enforce/playbooks/5xx/) for troubleshooting information.
    EOT

    links {
      display_name = "Playbook"
      url          = "https://wiki.inky.wtf/docs/teams/engineering/enforce/playbooks/5xx/"
    }
  }

  notification_channels = length(var.notification_channels) != 0 ? var.notification_channels : local.slack

  enabled = "true"
  project = var.project_id
}

moved {
  from = google_logging_metric.cloud-run-scaling-failure
  to   = google_logging_metric.cloud-run-scaling-failure[0]
}

resource "google_logging_metric" "cloud-run-scaling-failure" {
  count = var.squad == "" ? 1 : 0

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
  count = var.global_only_alerts ? 0 : 1

  # In the absence of data, incident will auto-close after an hour
  alert_strategy {
    auto_close = "3600s"

    notification_rate_limit {
      period = "3600s" // re-alert hourly if condition still valid.
    }
  }

  display_name = "Cloud Run scaling issue ${local.name}"
  combiner     = "OR"

  conditions {
    display_name = "Cloud Run scaling issue ${local.name}"

    condition_matched_log {
      filter = <<EOT
        resource.type="cloud_run_revision"
        log_name="projects/${var.project_id}/logs/run.googleapis.com%2Frequests"
        severity=ERROR
        textPayload:"The request was aborted because there was no available instance."
        ${var.scaling_issue_filter}
        ${local.squad_log_filter}
      EOT

      label_extractors = {
        "revision_name" = "EXTRACT(resource.labels.revision_name)"
        "location"      = "EXTRACT(resource.labels.location)"
      }
    }
  }

  notification_channels = length(var.notification_channels) != 0 ? var.notification_channels : local.slack

  enabled = var.enable_scaling_alerts
  project = var.project_id
}

moved {
  from = google_logging_metric.cloud-run-failed-req
  to   = google_logging_metric.cloud-run-failed-req[0]
}

resource "google_logging_metric" "cloud-run-failed-req" {
  count = var.squad == "" ? 1 : 0

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
  # In the absence of data, incident will auto-close after an hour
  alert_strategy {
    auto_close = "3600s"

    notification_rate_limit {
      period = "3600s" // re-alert hourly if condition still valid.
    }
  }

  display_name = "Cloud Run failed request ${local.name}"
  combiner     = "OR"

  conditions {
    display_name = "Cloud Run failed request ${local.name}"

    condition_matched_log {
      filter = <<EOT
        resource.type="cloud_run_revision"
        log_name="projects/${var.project_id}/logs/run.googleapis.com%2Frequests"
        severity=ERROR
        textPayload:"The request failed because either the HTTP response was malformed or connection to the instance had an error."
        ${var.failed_req_filter}
        ${local.squad_log_filter}
      EOT

      label_extractors = {
        "revision_name" = "EXTRACT(resource.labels.revision_name)"
        "location"      = "EXTRACT(resource.labels.location)"
      }
    }
  }

  notification_channels = length(var.notification_channels) != 0 ? var.notification_channels : local.slack

  enabled = var.enable_scaling_alerts
  project = var.project_id
}

resource "google_monitoring_alert_policy" "pubsub_dead_letter_queue_messages" {
  count = var.squad == "" ? 1 : 0

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

resource "google_monitoring_alert_policy" "cloudrun_timeout" {
  count = var.global_only_alerts ? 0 : 1

  # In the absence of data, incident will auto-close after an hour
  alert_strategy {
    auto_close = "3600s"

    notification_rate_limit {
      period = "3600s" // re-alert once an hour if condition still valid.
    }
  }

  display_name = "Timeout Alert ${local.name}"
  combiner     = "OR"

  conditions {
    display_name = "Timeout Alert ${local.name}"

    condition_matched_log {
      filter = <<EOT
        log_name="projects/${var.project_id}/logs/run.googleapis.com%2Frequests"
        severity=ERROR
        textPayload="The request has been terminated because it has reached the maximum request timeout. To change this limit, see https://cloud.google.com/run/docs/configuring/request-timeout"
        ${var.timeout_filter}
        ${local.squad_log_filter}
        -resource.labels.service_name:"-ing-vuln"
      EOT

      label_extractors = {
        "service_name"  = "EXTRACT(resource.labels.service_name)"
        "revision_name" = "EXTRACT(resource.labels.revision_name)"
        "job_name"      = "EXTRACT(resource.labels.job_name)"
        "location"      = "EXTRACT(resource.labels.location)"
      }
    }
  }

  enabled = false

  notification_channels = length(var.notification_channels) != 0 ? var.notification_channels : local.slack
}

moved {
  from = google_logging_metric.cloudrun_timeout
  to   = google_logging_metric.cloudrun_timeout[0]
}

resource "google_logging_metric" "cloudrun_timeout" {
  count = var.squad == "" ? 1 : 0

  name   = "cloudrun_timeout"
  filter = <<EOT
    resource.type="cloud_run_revision"
    log_name="projects/${var.project_id}/logs/run.googleapis.com%2Frequests"
    severity=ERROR
    textPayload="The request has been terminated because it has reached the maximum request timeout. To change this limit, see https://cloud.google.com/run/docs/configuring/request-timeout"
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

moved {
  from = google_logging_metric.dockerhub_ratelimit
  to   = google_logging_metric.dockerhub_ratelimit[0]
}

resource "google_logging_metric" "dockerhub_ratelimit" {
  count = var.squad == "" ? 1 : 0

  name   = "dockerhub_ratelimit"
  filter = <<EOT
    (resource.type="cloud_run_job" OR resource.type="cloud_run_revision")
    log_name="projects/${var.project_id}/logs/run.googleapis.com%2Fstderr"
    severity>=WARNING
    textPayload:"You have reached your pull rate limit. You may increase the limit by authenticating and upgrading: https://www.docker.com/increase-rate-limit"
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
    labels {
      key         = "job_name"
      value_type  = "STRING"
      description = "name of job."
    }
  }

  label_extractors = {
    "location"     = "EXTRACT(resource.labels.location)"
    "service_name" = "EXTRACT(resource.labels.service_name)"
    "job_name"     = "EXTRACT(resource.labels.job_name)"
  }
}

moved {
  from = google_logging_metric.github_ratelimit
  to   = google_logging_metric.github_ratelimit[0]
}

resource "google_logging_metric" "github_ratelimit" {
  count = var.squad == "" ? 1 : 0

  name   = "github_ratelimit"
  filter = <<EOT
    (resource.type="cloud_run_job" OR resource.type="cloud_run_revision")
    log_name="projects/${var.project_id}/logs/run.googleapis.com%2Fstderr"
    severity>=WARNING
    textPayload:"You have exceeded a secondary rate limit and have been temporarily blocked from content creation"
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
    labels {
      key         = "job_name"
      value_type  = "STRING"
      description = "name of job."
    }
  }

  label_extractors = {
    "location"     = "EXTRACT(resource.labels.location)"
    "service_name" = "EXTRACT(resource.labels.service_name)"
    "job_name"     = "EXTRACT(resource.labels.job_name)"
  }
}

moved {
  from = google_logging_metric.r2_same_ratelimit
  to   = google_logging_metric.r2_same_ratelimit[0]
}

resource "google_logging_metric" "r2_same_ratelimit" {
  count = var.squad == "" ? 1 : 0

  name   = "r2_same_ratelimit"
  filter = <<EOT
    (resource.type="cloud_run_job" OR resource.type="cloud_run_revision")
    log_name="projects/${var.project_id}/logs/run.googleapis.com%2Fstderr"
    severity>=WARNING
    textPayload:"Reduce your concurrent request rate for the same object"
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
    labels {
      key         = "job_name"
      value_type  = "STRING"
      description = "name of job."
    }
  }

  label_extractors = {
    "location"     = "EXTRACT(resource.labels.location)"
    "service_name" = "EXTRACT(resource.labels.service_name)"
    "job_name"     = "EXTRACT(resource.labels.job_name)"
  }
}
