module "received-events" {
  source = "../widgets/xy"
  title  = "Events Received"
  filter = [
    "resource.type=\"pubsub_subscription\"",
    "metric.type=\"pubsub.googleapis.com/subscription/push_request_count\"",
    "resource.label.\"subscription_id\"=monitoring.regex.full_match(\"${var.subscription_prefix}.*\")",
  ]
  group_by_fields = ["resource.label.\"subscription_id\""]
  primary_align   = "ALIGN_MEAN"
  primary_reduce  = "REDUCE_NONE"
}

module "oldest-unacked" {
  source = "../widgets/xy"
  title  = "Oldest unacked message age"
  filter = [
    "resource.type=\"pubsub_subscription\"",
    "metric.type=\"pubsub.googleapis.com/subscription/oldest_unacked_message_age\"",
    "resource.label.\"subscription_id\"=monitoring.regex.full_match(\"${var.subscription_prefix}.*\")",
  ]
  group_by_fields = ["resource.label.\"subscription_id\""]
  primary_align   = "ALIGN_MAX"
  primary_reduce  = "REDUCE_NONE"
}

module "undelivered" {
  source = "../widgets/xy"
  title  = "Undelivered messages"
  filter = [
    "resource.type=\"pubsub_subscription\"",
    "metric.type=\"pubsub.googleapis.com/subscription/num_undelivered_messages\"",
    "resource.label.\"subscription_id\"=monitoring.regex.full_match(\"${var.subscription_prefix}.*\")",
  ]
  group_by_fields = ["resource.label.\"subscription_id\""]
  primary_align   = "ALIGN_MAX"
  primary_reduce  = "REDUCE_NONE"
}

module "push-latency" {
  source = "../widgets/latency"
  title  = "Push latency"
  filter = [
    "resource.type=\"pubsub_subscription\"",
    "metric.type=\"pubsub.googleapis.com/subscription/push_request_latencies\"",
    "resource.label.\"subscription_id\"=monitoring.regex.full_match(\"${var.subscription_prefix}.*\")",
  ]
  group_by_fields = ["resource.label.\"subscription_id\""]
}

resource "google_monitoring_dashboard" "dashboard" {
  dashboard_json = jsonencode({
    displayName = "Subscriptions: ${var.subscription_prefix}",
    gridLayout = {
      columns = 3,
      widgets = [
        module.received-events.tile,
        module.oldest-unacked.tile,
        module.undelivered.tile,
        module.push-latency.tile,
      ]
    }
  })
}
