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

variable "raw_filter" {
  description = "Raw PubSub filter to apply, ignores other variables. https://cloud.google.com/pubsub/docs/subscription-message-filter#filtering_syntax"
  type        = string
  default     = ""
}

variable "filter" {
  description = <<EOD
A Knative Trigger-style filter over the cloud event attributes.

This is normally used to filter relevant event types, for example:

  { "type" : "dev.chainguard.foo" }

In this case, only events with a type attribute of "dev.chainguard.foo" will be delivered.
EOD
  type        = map(string)
  default     = {}
}

variable "filter_prefix" {
  description = <<EOD
A Knative Trigger-style filter over the cloud event attribute prefixes.

This can be used to filter relevant event types, for example:

  { "type" : "dev.chainguard." }

In this case, any event with a type attribute that starts with "dev.chainguard." will be delivered.
EOD
  type        = map(string)
  default     = {}
}

variable "filter_has_attributes" {
  description = <<EOD
A Knative Trigger-style filter over the cloud event attribute prefixes.

This can be used to filter on the presence of an event attribute, for example:

  ["location"]

In this case, any event with a type attribute of "location" will be delivered.
EOD
  type        = list(string)
  default     = []
}

variable "filter_not_has_attributes" {
  description = <<EOD
A Knative Trigger-style filter over the cloud event attribute prefixes.

This can be used to filter on the absence of an event attribute, for example:

  ["location"]

In this case, any event with a type attribute of "location" will NOT be delivered.
EOD
  type        = list(string)
  default     = []
}

variable "max_delivery_attempts" {
  description = "The maximum number of delivery attempts for any event."
  type        = number
  default     = 20
}

variable "minimum_backoff" {
  description = "The minimum delay between consecutive deliveries of a given message."
  type        = number
  default     = 10
}

variable "maximum_backoff" {
  description = "The maximum delay between consecutive deliveries of a given message."
  type        = number
  default     = 600
}

variable "ack_deadline_seconds" {
  description = "The deadline for acking a message."
  type        = number
  default     = 300
}

variable "require_team" {
  description = "Whether to require team variable to be specified"
  type        = bool
  default     = true
}

variable "team" {
  description = "team label to apply to the service."
  type        = string
  default     = ""

  validation {
    condition     = !var.require_team || var.team != ""
    error_message = "team needs to specified or disable check by setting require_team = false"
  }
}

variable "allowed_persistence_regions" {
  description = "The list of allowed persistence regions for the dead-letter topic."
  type        = list(string)
  default     = []
}

variable "product" {
  description = "Product label to apply to the service."
  type        = string
  default     = "unknown"
}
