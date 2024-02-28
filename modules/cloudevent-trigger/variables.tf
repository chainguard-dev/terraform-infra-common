variable "project_id" {
  type = string
}

variable "name" {
  type = string
}

variable "broker" {
  description = "The name of the pubsub topic we are using as a broker."
  type        = string
}

variable "filter" {
  description = <<EOD
A Knative Trigger-style filter over the cloud event attributes.

This is normally used to filter relevant event types:

  { "type" : "dev.chainguard.foo" }

EOD
  type        = map(string)
  default     = {}
}

variable "filter_prefix" {
  description = <<EOD
A Knative Trigger-style filter over the cloud event attribute prefixes.

If an event may have a "source" attribute "foo.bar" or "foo.baz" and the filter is

  { source = "foo." }

then both events will be delivered to the service.
EOD
  type        = map(string)
  default     = {}
}

variable "private-service" {
  description = "The private cloud run service that is subscribing to these events."
  type = object({
    name   = string
    region = string
  })
}

variable "max_delivery_attempts" {
  description = "The maximum number of delivery attempts for any event."
  type        = number
  default     = 5
}

variable "notification_channels" {
  description = "List of notification channels to alert."
  type        = list(string)
}
