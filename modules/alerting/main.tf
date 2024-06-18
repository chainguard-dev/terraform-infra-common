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

  notification_channels = var.notification_channels

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

  notification_channels = var.notification_channels
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

  notification_channels = var.notification_channels

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

  notification_channels = var.notification_channels

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

  notification_channels = var.notification_channels

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
    condition_monitoring_query_language {
      duration = "0s"
      query    = <<EOT
        fetch cloud_run_revision
        | metric 'run.googleapis.com/request_count'
        | group_by 5m, [value_request_count_aggregate: aggregate(value.request_count)]
        | every 5m
        | { group_by [metric.response_code_class, resource.service_name],
              [response_code_count_aggregate: aggregate(value_request_count_aggregate)]
            | filter (metric.response_code_class = '5xx')
          ; group_by [resource.service_name],
              [value_request_count_aggregate_aggregate:
                 aggregate(value_request_count_aggregate)] }
        | join
        | value [response_code_ratio: val(0) / val(1)]
        | condition gt(val(), 0.2)
      EOT

      trigger {
        count = "1"
      }
    }

    display_name = "5xx failure rate above ${var.failure_rate_ratio_threshold}"
  }

  display_name = "5xx failure rate above ${var.failure_rate_ratio_threshold}"

  documentation {
    content = <<-EOT
    Please consult the playbook entry [here](https://wiki.inky.wtf/docs/teams/engineering/enforce/playbooks/5xx/) for troubleshooting information.
    EOT
  }

  enabled = "true"
  project = var.project_id

  notification_channels = var.notification_channels
}
