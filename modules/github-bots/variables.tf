variable "name" {
  description = "The name of the bot."
  type        = string
}

variable "project_id" {
  description = "Project ID to create resources in."
  type        = string
}

variable "regions" {
  description = "A map from region names to a network and subnetwork."
  type = map(object({
    network = string
    subnet  = string
  }))
}

variable "broker" {
  description = "A map from each of the input region names to the name of the Broker topic in that region."
  type        = map(string)
}

variable "containers" {
  description = "The containers to run in the service.  Each container will be run in each region."
  type = map(object({
    source = object({
      base_image  = optional(string, "cgr.dev/chainguard/static:latest-glibc@sha256:b00a88ca2a8136cdbd86a8aa834c3f69c17debb295a38055f7babfc2c9f9a02b")
      working_dir = string
      importpath  = string
    })
    args = optional(list(string), [])
    ports = optional(list(object({
      name           = optional(string, "http1")
      container_port = optional(number, 8080)
    })), [])
    resources = optional(
      object(
        {
          limits = optional(object(
            {
              cpu    = string
              memory = string
            }
          ), null)
          cpu_idle          = optional(bool, true)
          startup_cpu_boost = optional(bool, true)
        }
      ),
      {
        cpu_idle = true
      }
    )
    env = optional(list(object({
      name  = string
      value = optional(string)
      value_source = optional(object({
        secret_key_ref = object({
          secret  = string
          version = string
        })
      }), null)
    })), [])
    regional-env = optional(list(object({
      name  = string
      value = map(string)
    })), [])
    volume_mounts = optional(list(object({
      name       = string
      mount_path = string
    })), [])
  }))
}

variable "github-event" {
  description = "The GitHub event type to subscribe to."
  type        = string
}

variable "notification_channels" {
  description = "List of notification channels to alert."
  type        = list(string)
}

variable "raw_filter" {
  description = "Raw PubSub filter to apply, ignores other variables. https://cloud.google.com/pubsub/docs/subscription-message-filter#filtering_syntax"
  type        = string
  default     = ""
}

variable "extra_filter" {
  type        = map(string)
  default     = {}
  description = "Optional additional filters to include."
}

variable "extra_filter_prefix" {
  type        = map(string)
  default     = {}
  description = "Optional additional prefixes for filtering events."
}

variable "extra_filter_has_attributes" {
  type        = list(string)
  default     = []
  description = "Optional additional attributes to check for presence."
}

variable "extra_filter_not_has_attributes" {
  type        = list(string)
  default     = []
  description = "Optional additional prefixes to check for presence."
}

variable "enable_profiler" {
  type        = bool
  default     = false
  description = "Enable cloud profiler."
}

variable "deletion_protection" {
  type        = bool
  description = "Whether to enable delete protection for the service."
  default     = true
}

variable "service_account_email" {
  description = "The email of the service account being authorized to invoke the private Cloud Run service. If empty, a service account will be created and used."
  type        = string
  default     = ""
}

variable "labels" {
  description = "Labels to apply to the service."
  type        = map(string)
  default     = {}
}



variable "team" {
  description = "Team label to apply to resources (replaces deprecated 'squad')."
  type        = string
  default     = ""
}

variable "product" {
  description = "Product label to apply to the service."
  type        = string
  default     = "unknown"
}
