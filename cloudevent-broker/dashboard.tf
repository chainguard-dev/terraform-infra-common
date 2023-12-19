module "sent-events" {
  source = "../dashboard/tiles/xy"
  title  = "Events Sent"
  filter = [
    "resource.type=\"pubsub_topic\"",
    "metric.type=\"pubsub.googleapis.com/topic/send_request_count\"",
  ]
  group_by_fields = ["resource.label.\"topic_id\""]
  primary_align   = "ALIGN_MEAN"
  primary_reduce  = "REDUCE_NONE"
}

module "received-events" {
  source = "../dashboard/tiles/xy"
  title  = "Events Received"
  filter = [
    "resource.type=\"pubsub_subscription\"",
    "metric.type=\"pubsub.googleapis.com/subscription/push_request_count\"",
  ]
  group_by_fields = ["resource.label.\"subscription_id\""]
  primary_align   = "ALIGN_MEAN"
  primary_reduce  = "REDUCE_NONE"
}

module "oldest-unacked" {
  source = "../dashboard/tiles/xy"
  title  = "Oldest unacked message age"
  filter = [
    "resource.type=\"pubsub_subscription\"",
    "metric.type=\"pubsub.googleapis.com/subscription/oldest_unacked_message_age\"",
  ]
  group_by_fields = ["resource.label.\"subscription_id\""]
  primary_align   = "ALIGN_MAX"
  primary_reduce  = "REDUCE_NONE"
}

module "undelivered" {
  source = "../dashboard/tiles/xy"
  title  = "Undelivered messages"
  filter = [
    "resource.type=\"pubsub_subscription\"",
    "metric.type=\"pubsub.googleapis.com/subscription/num_undelivered_messages\"",
  ]
  group_by_fields = ["resource.label.\"subscription_id\""]
  primary_align   = "ALIGN_MAX"
  primary_reduce  = "REDUCE_NONE"
}

module "undelivered" {
  source = "../dashboard/tiles/xy"
  title  = "Undelivered messages"
  filter = [
    "resource.type=\"pubsub_subscription\"",
    "metric.type=\"pubsub.googleapis.com/subscription/num_undelivered_messages\"",
  ]
  group_by_fields = ["resource.label.\"subscription_id\""]
  primary_align   = "ALIGN_MAX"
  primary_reduce  = "REDUCE_NONE"
}

module "push-latency" {
  source = "../dashboard/tiles/latency"
  title  = "Push latency"
  filter = [
    "resource.type=\"pubsub_subscription\"",
    "metric.type=\"pubsub.googleapis.com/subscription/push_request_latencies\"",
  ]
  group_by_fields = ["resource.label.\"subscription_id\""]

}

resource "google_monitoring_dashboard" "dashboard" {
  project = var.project_id

  dashboard_json = jsonencode({
    displayName = "Broker: ${var.name}",
    gridLayout = {
      columns = 3,
      widgets = [
        module.sent-events.tile,
        module.received-events.tile,
        module.oldest-unacked.tile,
        module.undelivered.tile,
        module.push-latency.tile
      ]
    }
  })
}
