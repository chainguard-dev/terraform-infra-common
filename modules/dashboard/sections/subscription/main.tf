variable "title" { type = string }
variable "subscription_prefix" { type = string }
variable "collapsed" { default = false }
variable "notification_channels" {
  type = list(string)
}
variable "alert_threshold" {
  type    = number
  default = 50000
}


module "width" { source = "../width" }

resource "google_monitoring_alert_policy" "pubsub_unacked_messages" {
  // Close after 7 days
  alert_strategy {
    auto_close = "604800s"
  }

  combiner = "OR"

  conditions {
    condition_threshold {
      aggregations {
        alignment_period   = "300s"
        per_series_aligner = "ALIGN_MEAN"
      }

      comparison = "COMPARISON_GT"
      duration   = "0s"
      filter     = "resource.type = \"pubsub_subscription\" AND metric.type = \"pubsub.googleapis.com/subscription/num_unacked_messages_by_region\" AND metadata.system_labels.name = monitoring.regex.full_match(\"${var.subscription_prefix}-[^.]*\")"

      trigger {
        count = "1"
      }

      threshold_value = var.alert_threshold
    }

    display_name = "${var.title}: Unacked messages above ${var.alert_threshold}"
  }
  display_name = "${var.title}: Unacked messages above ${var.alert_threshold}"

  enabled = "true"

  notification_channels = var.notification_channels
}

module "unacked-messages-alert" {
  source     = "../../widgets/alert"
  title      = google_monitoring_alert_policy.pubsub_unacked_messages.display_name
  alert_name = google_monitoring_alert_policy.pubsub_unacked_messages.name
}

module "received-events" {
  source = "../../widgets/xy"
  title  = "Events Pushed"
  filter = [
    "resource.type=\"pubsub_subscription\"",
    "metric.type=\"pubsub.googleapis.com/subscription/push_request_count\"",
    "resource.label.\"subscription_id\"=monitoring.regex.full_match(\"${var.subscription_prefix}-[^-]*\")",
  ]
  group_by_fields = [
    "resource.label.\"subscription_id\"",
    "metric.label.\"response_class\""
  ]
  primary_align  = "ALIGN_MEAN"
  primary_reduce = "REDUCE_NONE"
}

module "push-latency" {
  source = "../../widgets/latency"
  title  = "Push latency"
  filter = [
    "resource.type=\"pubsub_subscription\"",
    "metric.type=\"pubsub.googleapis.com/subscription/push_request_latencies\"",
    "resource.label.\"subscription_id\"=monitoring.regex.full_match(\"${var.subscription_prefix}-[^-]*\")",
  ]
  group_by_fields = ["resource.label.\"subscription_id\""]
}

module "oldest-unacked" {
  source = "../../widgets/xy"
  title  = "Oldest unacked message age"
  filter = [
    "resource.type=\"pubsub_subscription\"",
    "metric.type=\"pubsub.googleapis.com/subscription/oldest_unacked_message_age\"",
    "resource.label.\"subscription_id\"=monitoring.regex.full_match(\"${var.subscription_prefix}-[^-]*\")",
  ]
  group_by_fields = ["resource.label.\"subscription_id\""]
  primary_align   = "ALIGN_MAX"
  primary_reduce  = "REDUCE_NONE"
}

locals {
  columns = 3
  unit    = module.width.size / local.columns

  // https://www.terraform.io/language/functions/range
  // N columns, unit width each  ([0, unit, 2 * unit, ...])
  col = range(0, local.columns * local.unit, local.unit)

  tiles = [
    {
      yPos   = 0,
      xPos   = local.col[0],
      height = local.unit,
      width  = module.width.size,
      widget = module.unacked-messages-alert.widget,
    },
    {
      yPos   = local.unit,
      xPos   = local.col[0],
      height = local.unit,
      width  = local.unit,
      widget = module.received-events.widget,
    },
    {
      yPos   = local.unit,
      xPos   = local.col[1],
      height = local.unit,
      width  = local.unit,
      widget = module.push-latency.widget,
    },
    {
      yPos   = local.unit,
      xPos   = local.col[2],
      height = local.unit,
      width  = local.unit,
      widget = module.oldest-unacked.widget,
    }
  ]
}

module "collapsible" {
  source = "../collapsible"

  title     = var.title
  tiles     = local.tiles
  collapsed = var.collapsed
}

output "section" {
  value = module.collapsible.section
}
