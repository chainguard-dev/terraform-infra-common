variable "project_id" {
  type = string
}

variable "name" {
  type = string
}

variable "regions" {
  description = "A map from region names to a network and subnetwork.  A pub/sub topic and ingress service (publishing to the respective topic) will be created in each region, with the ingress service configured to egress all traffic via the specified subnetwork."
  type = map(object({
    network = string
    subnet  = string
  }))
}

variable "broker" {
  description = "A map from each of the input region names to the name of the Broker topic in that region."
  type        = map(string)
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

variable "job" {
  description = "The Cloud Run Job to invoke when an event is received."
  type = object({
    name            = string
    service_account = string
    // If unset, the job will be run in every region in var.regions.
    // If set, it must be a key in var.regions.
    region = optional(string, "")
    source = object({
      importpath  = string
      working_dir = string
      base_image  = optional(string, "")
    })
  })
}

variable "max_delivery_attempts" {
  description = "The maximum number of delivery attempts for any event."
  type        = number
  default     = 20
}

variable "notification_channels" {
  description = "List of notification channels to alert."
  type        = list(string)
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

variable "base_image" {
  type        = string
  default     = "cgr.dev/chainguard/static:latest-glibc"
  description = "The base image that will be used to build the Job's container image."
}

variable "execution_environment" {
  default     = ""
  type        = string
  description = "The execution environment to use for the job."
}

variable "max_retries" {
  default     = 3 # 3 retries is the default for Cloud Run jobs
  type        = number
  description = "The maximum number of times to retry the job."
}

variable "timeout" {
  default     = "600s" # 10 minutes is the default for Cloud Run jobs
  type        = string
  description = "The maximum amount of time in seconds to allow the job to run."
}

variable "cpu" {
  type        = string
  default     = "1000m"
  description = "The CPU limit for the job."
}

variable "memory" {
  type        = string
  default     = "512Mi"
  description = "The memory limit for the job."
}

variable "task_count" {
  type        = number
  default     = 1
  description = "The number of tasks to run. "
}

variable "parallelism" {
  type        = number
  default     = 1
  description = "The number of parallel jobs to run. Must be <= task_count"
  validation {
    condition     = var.parallelism <= var.task_count
    error_message = "parallelism must be less than or equal to task_count"
  }
}

variable "deletion_protection" {
  type        = bool
  description = "Whether to enable delete protection for the service."
  default     = true
}

variable "labels" {
  description = "Labels to apply to the job."
  type        = map(string)
  default     = {}
}

variable "require_squad" {
  description = "Whether to require squad variable to be specified"
  type        = bool
  default     = true
}

variable "squad" {
  description = "squad label to apply to the service."
  type        = string
  default     = ""

  validation {
    condition     = !var.require_squad || var.squad != ""
    error_message = "squad needs to specified or disable check by setting require_squad = false"
  }
}

variable "enable_otel_sidecar" {
  description = "Enable otel sidecar for metrics"
  type        = bool
  default     = false
}

variable "otel_collector_image" {
  type        = string
  default     = "chainguard/opentelemetry-collector-contrib:latest"
  description = "The otel collector image to use as a base. Must be on gcr.io or dockerhub."
}
