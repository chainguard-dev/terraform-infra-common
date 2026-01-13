variable "project_id" {
  description = "The GCP project ID"
  type        = string
}

variable "name" {
  description = "The base name for resources"
  type        = string
}

variable "regions" {
  description = "A map of regions to launch services in (see regional-go-service module for format)"
  type = map(object({
    network = string
    subnet  = string
  }))
}

variable "broker" {
  type        = map(string)
  description = "A map from each of the input region names to the name of the Broker topic in that region."
}

variable "filters" {
  description = <<EOD
A list of Knative Trigger-style filters over cloud event attributes.

Each filter is a map of attribute key-value pairs that must match exactly.
Multiple filters are combined with OR logic (any filter can match).

Examples:
  # Single event type
  filters = [
    { "type" = "dev.chainguard.github.pull_request" }
  ]

  # Multiple event types
  filters = [
    { "type" = "dev.chainguard.github.pull_request" },
    { "type" = "dev.chainguard.github.pull_request_review" }
  ]

  # Filter by type and action
  filters = [
    {
      "type"   = "dev.chainguard.github.pull_request"
      "action" = "opened"
    }
  ]
EOD
  type        = list(map(string))
  default     = []
}

variable "extension_key" {
  description = "The CloudEvent extension attribute to use as the workqueue key (e.g., pullrequesturl or issueurl)"
  type        = string
}

variable "workqueue" {
  description = "The workqueue to send events to"
  type = object({
    name = string
  })
}

variable "notification_channels" {
  description = "List of notification channels for alerts"
  type        = list(string)
}



variable "team" {
  description = "Team label to apply to resources (replaces deprecated 'squad')."
  type        = string
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

variable "product" {
  description = "Product label to apply to the service."
  type        = string
  default     = "unknown"
}

variable "deletion_protection" {
  description = "Whether to enable deletion protection for resources"
  type        = bool
  default     = true
}

variable "priority" {
  description = "Priority for workqueue items (higher values = higher priority)"
  type        = number
  default     = 0
}
