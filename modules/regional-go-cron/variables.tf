// Copyright 2026 Chainguard, Inc.
// SPDX-License-Identifier: Apache-2.0

variable "project_id" {
  type = string
}

variable "name" {
  type = string
}

variable "regions" {
  description = "A map from region names to a network and subnetwork. A job and scheduler will be created in each region."
  type = map(object({
    network = string
    subnet  = string
  }))
}

variable "regional-cronspec" {
  description = "Per-region cron schedule configuration. Must contain an entry for every key in var.regions."
  type = map(object({
    schedule  = string
    time_zone = optional(string, "America/New_York")
    paused    = optional(bool, false)
  }))
}

variable "invokers" {
  description = "Additional IAM members granted roles/run.invoker on the job, beyond the dedicated invoker service account."
  type        = list(string)
  default     = []
}

variable "egress" {
  type        = string
  description = "Which type of egress traffic to route through the VPC. ALL_TRAFFIC or PRIVATE_RANGES_ONLY."
  default     = "ALL_TRAFFIC"
}

variable "service_account" {
  type        = string
  description = "The service account as which each job task runs."
}

variable "deletion_protection" {
  type        = bool
  description = "Whether to enable delete protection on the Cloud Run Jobs."
  default     = true
}

variable "containers" {
  description = "The containers to run in each job task. Ports, probes, and cpu_idle are accepted for type compatibility with regional-go-service but are not used in job tasks."
  type = map(object({
    source = object({
      base_image  = optional(string, "cgr.dev/chainguard/static:latest-glibc@sha256:2fdfacc8d61164aa9e20909dceec7cc28b9feb66580e8e1a65b9f2443c53b61b")
      working_dir = string
      importpath  = string
      env         = optional(list(string), [])
    })
    command = optional(list(string), [])
    args    = optional(list(string), [])
    ports = optional(list(object({
      name           = optional(string, "h2c")
      container_port = number
    })), [])
    resources = optional(object({
      limits = optional(object({
        cpu    = string
        memory = string
      }), null)
      cpu_idle          = optional(bool)
      startup_cpu_boost = optional(bool, true)
    }), {})
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
    startup_probe  = optional(any)
    liveness_probe = optional(any)
  }))
  default = {}
}

variable "labels" {
  description = "Additional labels to apply to all resources."
  type        = map(string)
  default     = {}
}

variable "team" {
  description = "Team label to apply to resources."
  type        = string
}

variable "product" {
  description = "Product label to apply to resources."
  type        = string
  default     = "unknown"
}

variable "volumes" {
  description = "Volumes to make available to job task containers."
  type = list(object({
    name = string
    empty_dir = optional(object({
      medium     = optional(string, "MEMORY")
      size_limit = optional(string)
    }))
    secret = optional(object({
      secret = string
      items = list(object({
        version = string
        path    = string
      }))
    }))
    nfs = optional(object({
      server    = string
      path      = string
      read_only = optional(bool, true)
    }))
    gcs = optional(object({
      bucket        = string
      read_only     = optional(bool, true)
      mount_options = optional(list(string), [])
    }))
  }))
  default = []
}

variable "notification_channels" {
  description = "Notification channels for alerts."
  type        = list(string)
  default     = []
}

variable "execution_environment" {
  type    = string
  default = "EXECUTION_ENVIRONMENT_GEN2"
}

variable "launch_stage" {
  type    = string
  default = "GA"
}

variable "max_retries" {
  description = "Maximum number of times a task is retried on failure. 0 means no retries."
  type        = number
  default     = 0
}

variable "timeout" {
  description = "Maximum time allowed for a single task execution."
  type        = string
  default     = "600s"
}

variable "task_count" {
  type    = number
  default = 1
}

variable "parallelism" {
  type    = number
  default = 1
}

variable "enable_otel_sidecar" {
  type    = bool
  default = true
}

variable "otel_collector_image" {
  type    = string
  default = "chainguard/opentelemetry-collector-contrib:latest"
}

variable "otel_resources" {
  description = "Resources to add to the OpenTelemetry resource."
  type        = map(string)
  default     = {}
}

variable "success_alert_alignment_period_seconds" {
  description = "Alignment period for successful completion alert. 0 (default) to not create alert."
  type        = number
  default     = 0
  validation {
    condition     = var.success_alert_alignment_period_seconds <= 60 * 60 * 20
    error_message = "Alignment period must be less than or equal to 20h (in seconds). Note: When combined with a custom duration, the total alert horizon (alignment_period + duration) must be <= 25h per GCP limits."
  }
}

variable "success_alert_duration_seconds" {
  description = "How long the absence of successful executions must persist before the alert fires. If not set or 0, defaults to success_alert_alignment_period_seconds for backward compatibility."
  type        = number
  default     = 0
  validation {
    condition = var.success_alert_duration_seconds == 0 || (
      var.success_alert_duration_seconds >= 60 &&
      var.success_alert_duration_seconds <= 84600
    )
    error_message = "Duration must be either 0 (to use alignment period value) or between 60 seconds and 23.5 hours (GCP maximum)."
  }
}
