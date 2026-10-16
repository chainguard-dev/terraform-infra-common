locals {
  trampoline_logs_filter = <<-EOT
    resource.type="cloud_run_revision"
    resource.labels.service_name="${var.name}"
    severity>=WARNING
  EOT
  ingress_logs_filter    = <<-EOT
    resource.type="cloud_run_revision"
    resource.labels.service_name="${var.ingress.name}"
    severity>=WARNING
  EOT

  trampoline_logs_url   = "https://console.cloud.google.com/logs/query;query=${urlencode(local.trampoline_logs_filter)}?project=${var.project_id}"
  ingress_logs_url      = "https://console.cloud.google.com/logs/query;query=${urlencode(local.ingress_logs_filter)}?project=${var.project_id}"
  subscription_url      = "https://console.cloud.google.com/cloudpubsub/subscription/detail/${google_pubsub_subscription.this.name}?project=${var.project_id}"
  dead_letter_url       = "https://console.cloud.google.com/cloudpubsub/subscription/detail/${google_pubsub_subscription.dead-letter-pull-sub.name}?project=${var.project_id}"
  dashboard_url         = "https://console.cloud.google.com/monitoring/dashboards/builder/${basename(module.dashboard.id)}?project=${var.project_id}"
  dead_letter_pull_hint = "gcloud pubsub subscriptions pull ${google_pubsub_subscription.dead-letter-pull-sub.name} --project=${var.project_id} --limit=10"

  // Shared by every alert: where to look and what usually breaks.
  troubleshooting = <<-EOT
    ## Troubleshooting

    1. Open the [trampoline logs](${local.trampoline_logs_url}). `failed to deliver event` carries the broker's response. `failed to unmarshal notification` or `notification has no incident_id` mean it rejected a notification with 400. No request logs at all means Pub/Sub is not reaching the service.
    2. Open the [push subscription](${local.subscription_url}) metrics and break down "Push requests" by response class:
       - `remote_server_5xx`: the trampoline could not hand the event to the broker. Check the [broker ingress logs](${local.ingress_logs_url}) and whether `${var.ingress.name}` is serving.
       - `remote_server_4xx` with 401 or 403: Pub/Sub's push identity lost `roles/run.invoker` on `${var.name}`. With 400: the trampoline could not parse a notification.
       - `deadline_exceeded` or `unreachable`: the trampoline is hanging or not starting. Check its latest revision on the [dashboard](${local.dashboard_url}).
    3. If a recent deploy of `${var.name}` or `${var.ingress.name}` lines up with the start, roll it back.

    Once deliveries succeed, check the [dead-letter subscription](${local.dead_letter_url}) for incidents that were lost while it was broken: `${local.dead_letter_pull_hint}` (leave out `--auto-ack` to keep them). It keeps messages for 24 hours. Republishing them to the source topic also delivers them again to every other subscriber on it, so do that only if they tolerate duplicates.

    If the cause is outside this module, escalate to the team that owns it; this module belongs to `${var.team}`.
  EOT
}

resource "google_monitoring_alert_policy" "dead-letter" {
  project = var.project_id

  display_name = "Incident events dead-lettered: ${var.name}"
  severity     = "WARNING"
  combiner     = "OR"
  enabled      = true

  alert_strategy {
    auto_close = "3600s"
  }

  conditions {
    display_name = "Incident events dead-lettered: ${var.name}"

    // A topic metric rather than the subscription's dead_letter_message_count,
    // so publishing a test message to the dead-letter topic exercises the alert.
    condition_threshold {
      filter = <<-EOT
        resource.type="pubsub_topic"
        metric.type="pubsub.googleapis.com/topic/send_request_count"
        resource.label.topic_id="${google_pubsub_topic.dead-letter.name}"
      EOT

      aggregations {
        alignment_period     = "300s"
        per_series_aligner   = "ALIGN_DELTA"
        cross_series_reducer = "REDUCE_SUM"
      }

      comparison      = "COMPARISON_GT"
      threshold_value = 0
      duration        = "0s"

      trigger {
        count = 1
      }
    }
  }

  documentation {
    subject   = "Cloud Monitoring incidents dead-lettered by ${var.name}"
    mime_type = "text/markdown"
    content   = <<-EOT
      At least one Cloud Monitoring incident notification failed delivery ${var.max_delivery_attempts} times and was moved to the dead-letter topic `${google_pubsub_topic.dead-letter.name}`. That incident never reached the broker, so nothing subscribed to `dev.chainguard.gcp.monitoring.incident` saw it. Notifications that Cloud Monitoring sends straight to Slack, email or other channels are not affected.

      One dead-lettered incident on its own is usually a notification the trampoline rejected (HTTP 400). Several at once, or this alert together with "Incident events delivery failing", mean forwarding is broken.

      ${local.troubleshooting}
    EOT

    links {
      display_name = "Dead-letter subscription"
      url          = local.dead_letter_url
    }
    links {
      display_name = "Trampoline logs"
      url          = local.trampoline_logs_url
    }
    links {
      display_name = "Push subscription"
      url          = local.subscription_url
    }
  }

  notification_channels = var.notification_channels
}

resource "google_monitoring_alert_policy" "backlog" {
  project = var.project_id

  display_name = "Incident events delivery stuck: ${var.name}"
  severity     = "WARNING"
  combiner     = "OR"
  enabled      = true

  alert_strategy {
    auto_close = "3600s"
  }

  conditions {
    display_name = "Incident events delivery stuck: ${var.name}"

    // A healthy subscription stays under a minute, cold starts included. A
    // trampoline that hangs holds each attempt for the 300s ack deadline, so
    // this crosses 600s after two attempts, long before the message
    // dead-letters.
    condition_threshold {
      filter = <<-EOT
        resource.type="pubsub_subscription"
        metric.type="pubsub.googleapis.com/subscription/oldest_unacked_message_age"
        resource.label.subscription_id="${google_pubsub_subscription.this.name}"
      EOT

      aggregations {
        alignment_period   = "60s"
        per_series_aligner = "ALIGN_MAX"
      }

      comparison      = "COMPARISON_GT"
      threshold_value = 600
      duration        = "0s"

      trigger {
        count = 1
      }
    }
  }

  documentation {
    subject   = "Cloud Monitoring incidents stuck in ${google_pubsub_subscription.this.name}"
    mime_type = "text/markdown"
    content   = <<-EOT
      The oldest unacknowledged incident notification on the push subscription `${google_pubsub_subscription.this.name}` has been waiting more than 10 minutes. Normally the trampoline `${var.name}` forwards each one to the broker within a minute. Until this clears, subscribers to `dev.chainguard.gcp.monitoring.incident` get incidents late or not at all. Notifications that Cloud Monitoring sends straight to Slack, email or other channels are not affected.

      A trampoline that hangs instead of failing, or Pub/Sub backing off pushes after repeated failures, shows up here before anything is dead-lettered.

      ${local.troubleshooting}
    EOT

    links {
      display_name = "Push subscription"
      url          = local.subscription_url
    }
    links {
      display_name = "Trampoline logs"
      url          = local.trampoline_logs_url
    }
    links {
      display_name = "Dashboard"
      url          = local.dashboard_url
    }
  }

  notification_channels = var.notification_channels
}

resource "google_monitoring_alert_policy" "push-failures" {
  project = var.project_id

  display_name = "Incident events delivery failing: ${var.name}"
  severity     = "WARNING"
  combiner     = "OR"
  enabled      = true

  alert_strategy {
    auto_close = "3600s"
  }

  conditions {
    display_name = "Incident events delivery failing: ${var.name}"

    // Measured from Pub/Sub's side, so it also counts the 401, 403 and
    // timeouts that never reach the trampoline. One notification that fails
    // every attempt makes exactly max_delivery_attempts failed pushes, so it
    // raises only the dead-letter alert. More failures than that in the
    // window mean more than one notification is failing.
    condition_threshold {
      filter = <<-EOT
        resource.type="pubsub_subscription"
        metric.type="pubsub.googleapis.com/subscription/push_request_count"
        resource.label.subscription_id="${google_pubsub_subscription.this.name}"
        metric.label.response_class!="ack"
      EOT

      aggregations {
        alignment_period     = "600s"
        per_series_aligner   = "ALIGN_DELTA"
        cross_series_reducer = "REDUCE_SUM"
      }

      comparison      = "COMPARISON_GT"
      threshold_value = var.max_delivery_attempts
      duration        = "0s"

      trigger {
        count = 1
      }
    }
  }

  documentation {
    subject   = "Cloud Monitoring incident deliveries to ${var.name} failing"
    mime_type = "text/markdown"
    content   = <<-EOT
      Pub/Sub pushes from `${google_pubsub_subscription.this.name}` to the trampoline `${var.name}` failed more than ${var.max_delivery_attempts} times in 10 minutes, so more than one notification is failing. Each failed incident is retried with backoff and dead-lettered after ${var.max_delivery_attempts} attempts, so incidents are reaching subscribers to `dev.chainguard.gcp.monitoring.incident` late, and some may be lost. Notifications that Cloud Monitoring sends straight to Slack, email or other channels are not affected.

      The trampoline returns 500 when the broker ingress rejects or cannot take an event, so an ingress outage raises this alert alongside "Incident events dead-lettered".

      ${local.troubleshooting}
    EOT

    links {
      display_name = "Push subscription"
      url          = local.subscription_url
    }
    links {
      display_name = "Trampoline logs"
      url          = local.trampoline_logs_url
    }
    links {
      display_name = "Broker ingress logs"
      url          = local.ingress_logs_url
    }
  }

  notification_channels = var.notification_channels
}
