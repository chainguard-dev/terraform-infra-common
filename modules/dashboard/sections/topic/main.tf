variable "title" { type = string }
variable "topic_prefix" { type = string }
variable "collapsed" { default = false }

module "width" { source = "../width" }

module "sent-events" {
  source = "../../widgets/xy"
  title  = "Events Published"
  filter = [
    "resource.type=\"pubsub_topic\"",
    "metric.type=\"pubsub.googleapis.com/topic/send_request_count\"",
    "resource.label.\"topic_id\"=monitoring.regex.full_match(\"${var.topic_prefix}-.*\")",
  ]
  group_by_fields = ["resource.label.\"topic_id\""]
  primary_align   = "ALIGN_MEAN"
  primary_reduce  = "REDUCE_NONE"
}

module "send-latency" {
  source = "../../widgets/latency"
  title  = "Publish latency"
  filter = [
    "resource.type=\"pubsub_topic\"",
    "metric.type=\"pubsub.googleapis.com/topic/send_request_latencies\"",
    "resource.label.\"topic_id\"=monitoring.regex.full_match(\"${var.topic_prefix}-.*\")",
  ]
  group_by_fields = ["resource.label.\"topic_id\""]
}

module "topic-oldest-unacked" {
  source = "../../widgets/xy"
  title  = "Oldest unacked message age (topic)"
  filter = [
    "resource.type=\"pubsub_topic\"",
    "metric.type=\"pubsub.googleapis.com/topic/oldest_unacked_message_age_by_region\"",
    "resource.label.\"topic_id\"=monitoring.regex.full_match(\"${var.topic_prefix}-.*\")",
  ]
  group_by_fields = ["resource.label.\"topic_id\""]
  primary_align   = "ALIGN_MAX"
  primary_reduce  = "REDUCE_NONE"
}

module "received-events" {
  source = "../../widgets/xy"
  title  = "Events Pushed"
  filter = [
    "resource.type=\"pubsub_subscription\"",
    "metric.type=\"pubsub.googleapis.com/subscription/push_request_count\"",
    "metadata.system_labels.\"topic_id\"=monitoring.regex.full_match(\"${var.topic_prefix}-.*\")",
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
    "metadata.system_labels.\"topic_id\"=monitoring.regex.full_match(\"${var.topic_prefix}-.*\")",
  ]
  group_by_fields = ["resource.label.\"subscription_id\""]
}

module "oldest-unacked" {
  source = "../../widgets/xy"
  title  = "Oldest unacked message age"
  filter = [
    "resource.type=\"pubsub_subscription\"",
    "metric.type=\"pubsub.googleapis.com/subscription/oldest_unacked_message_age\"",
    "metadata.system_labels.\"topic_id\"=monitoring.regex.full_match(\"${var.topic_prefix}-.*\")",
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

  tiles = [{
    yPos   = 0,
    xPos   = local.col[0],
    height = local.unit,
    width  = local.unit,
    widget = module.sent-events.widget,
    },
    {
      yPos   = 0,
      xPos   = local.col[1],
      height = local.unit,
      width  = local.unit,
      widget = module.send-latency.widget,
    },
    {
      yPos   = 0,
      xPos   = local.col[2],
      height = local.unit,
      width  = local.unit,
      widget = module.topic-oldest-unacked.widget,
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
  }]
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
