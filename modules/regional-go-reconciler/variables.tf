/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

variable "project_id" {
  type = string
}

variable "name" {
  type = string
}

variable "regions" {
  description = "A map from region names to a network and subnetwork.  A service will be created in each region configured to egress the specified traffic via the specified subnetwork."
  type = map(object({
    network = string
    subnet  = string
  }))
}

// Workqueue-specific variables

variable "max-retry" {
  description = "The maximum number of times a task will be retried before being moved to the dead-letter queue. Set to 0 for unlimited retries."
  type        = number
  default     = 100
}

variable "concurrent-work" {
  description = "The amount of concurrent work to dispatch at a given time."
  type        = number
  default     = 20
}

variable "multi_regional_location" {
  description = "The multi-regional location for the global workqueue bucket. Options: US, EU, ASIA."
  type        = string
  default     = "US"
  validation {
    condition     = contains(["US", "EU", "ASIA"], var.multi_regional_location)
    error_message = "multi_regional_location must be one of: US, EU, ASIA."
  }
}

// Service-specific variables

variable "egress" {
  type        = string
  description = <<EOD
Which type of egress traffic to send through the VPC.

- ALL_TRAFFIC sends all traffic through regional VPC network. This should be used if service is not expected to egress to the Internet.
- PRIVATE_RANGES_ONLY sends only traffic to private IP addresses through regional VPC network
EOD
  default     = "ALL_TRAFFIC"
}

variable "service_account" {
  type        = string
  description = "The service account as which to run the reconciler service."
}

variable "deletion_protection" {
  type        = bool
  description = "Whether to enable delete protection for the service."
  default     = true
}

variable "containers" {
  description = "The containers to run in the service.  Each container will be run in each region."
  type = map(object({
    source = object({
      base_image  = optional(string, "cgr.dev/chainguard/static:latest-glibc@sha256:b00a88ca2a8136cdbd86a8aa834c3f69c17debb295a38055f7babfc2c9f9a02b")
      working_dir = string
      importpath  = string
      env         = optional(list(string), [])
    })
    args = optional(list(string), [])
    ports = optional(list(object({
      name           = optional(string, "h2c")
      container_port = number
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
          cpu_idle          = optional(bool)
          startup_cpu_boost = optional(bool, true)
        }
      ),
      {}
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
    regional-cpu-idle = optional(map(bool), {})
    volume_mounts = optional(list(object({
      name       = string
      mount_path = string
    })), [])
    startup_probe = optional(object({
      initial_delay_seconds = optional(number)
      timeout_seconds       = optional(number, 240)
      period_seconds        = optional(number, 240)
      failure_threshold     = optional(number, 1)
      tcp_socket = optional(object({
        port = optional(number)
      }), null)
      grpc = optional(object({
        port    = optional(number)
        service = optional(string)
      }), null)
    }), null)
    liveness_probe = optional(object({
      initial_delay_seconds = optional(number)
      timeout_seconds       = optional(number)
      period_seconds        = optional(number)
      failure_threshold     = optional(number)
      http_get = optional(object({
        path = optional(string)
        http_headers = optional(list(object({
          name  = string
          value = string
        })), [])
      }), null)
      grpc = optional(object({
        port    = optional(number)
        service = optional(string)
      }), null)
    }), null)
  }))
  default = {}
}

// Common variables

variable "labels" {
  description = "Additional labels to add to all resources."
  type        = map(string)
  default     = {}
}

variable "team" {
  description = "Team label to apply to resources (replaces deprecated 'squad')."
  type        = string
  default     = ""
}

variable "product" {
  description = "The product that this service belongs to."
  type        = string
  default     = ""
}

variable "scaling" {
  description = "The scaling configuration for the service."
  type = object({
    min_instances                    = optional(number, 0)
    max_instances                    = optional(number, 100)
    max_instance_request_concurrency = optional(number, 1000)
  })
  default = {}
}

variable "volumes" {
  description = "The volumes to attach to the service."
  type = list(object({
    name = string
    empty_dir = optional(object({
      medium     = optional(string, "MEMORY")
      size_limit = optional(string, "1Gi")
    }), null)
    csi = optional(object({
      driver = string
      volume_attributes = optional(object({
        bucketName = string
      }), null)
    }), null)
  }))
  default = []
}

variable "regional-volumes" {
  description = "The volumes to make available to the containers in the service for mounting."
  type = list(object({
    name = string
    gcs = optional(map(object({
      bucket        = string
      read_only     = optional(bool, true)
      mount_options = optional(list(string), [])
    })), {})
    nfs = optional(map(object({
      server    = string
      path      = string
      read_only = optional(bool, true)
    })), {})
  }))
  default = []
}

variable "enable_profiler" {
  description = "Enable continuous profiling for the service.  This has a small performance impact, which shouldn't matter for production services."
  type        = bool
  default     = true
}

variable "otel_resources" {
  description = "Resources to add to the OpenTelemetry resource."
  type        = map(string)
  default     = {}
}

variable "request_timeout_seconds" {
  description = "The request timeout for the service in seconds."
  type        = number
  default     = 300
}

variable "execution_environment" {
  description = "The execution environment for the service (options: EXECUTION_ENVIRONMENT_GEN1, EXECUTION_ENVIRONMENT_GEN2)."
  type        = string
  default     = "EXECUTION_ENVIRONMENT_GEN2"
}

variable "notification_channels" {
  description = "The channels to send notifications to. List of channel IDs"
  type        = list(string)
  default     = []
}

variable "workqueue_cpu_idle" {
  description = "Set to false for a region in order to use instance-based billing for workqueue services (dispatcher and receiver). Defaults to true. To control reconciler cpu_idle, use the 'regional-cpu-idle' field in the 'containers' variable."
  type        = map(map(bool))
  default = {
    "dispatcher" = {}
    "receiver"   = {}
  }
}

variable "slo" {
  description = "Configuration for setting up SLO for the cloud run service"
  type = object({
    enable          = optional(bool, false)
    enable_alerting = optional(bool, false)
    success = optional(object(
      {
        multi_region_goal = optional(number, 0.999)
        per_region_goal   = optional(number, 0.999)
      }
    ), null)
    monitor_gclb = optional(bool, false)
  })
  default = {}
}
