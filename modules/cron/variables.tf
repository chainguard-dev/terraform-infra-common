variable "name" {
  description = "Name to prefix to created resources."
}

variable "project_id" {
  type        = string
  description = "The project that will host the cron job."
}

variable "region" {
  default     = "us-east4"
  description = "The region to run the job."
}

variable "schedule" {
  description = "The cron schedule on which to run the job."
}

variable "base_image" {
  type        = string
  default     = "cgr.dev/chainguard/static:latest-glibc@sha256:a301031ffd4ed67f35ca7fa6cf3dad9937b5fa47d7493955a18d9b4ca5412d1a"
  description = "The base image that will be used to build the container image."
}

variable "repository" {
  type        = string
  default     = ""
  description = "Container repository to publish images to."
}

variable "service_account" {
  type        = string
  description = "The email address of the service account to run the service as, and to invoke the job as."
}

variable "importpath" {
  type        = string
  description = "The import path that contains the cron application. Leave empty to run the unmodified base image as the application: for example, when running an `apko`-built image. This works by skipping the `ko` build and just use the base image directly in the cron job. A digest must be specified in this case."
  default     = ""
}

variable "ko_build_env" {
  type        = list(string)
  description = "A list of custom environment variables to pass to the ko build."
  default     = []
}

variable "working_dir" {
  type        = string
  description = "The working directory that contains the importpath."
}

variable "env" {
  default     = {}
  description = "A map of custom environment variables (e.g. key=value)"
}

variable "secret_env" {
  default     = {}
  description = "A map of secrets to mount as environment variables from Google Secrets Manager (e.g. secret_key=secret_name)"
}

variable "execution_environment" {
  default     = "EXECUTION_ENVIRONMENT_GEN2"
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

variable "vpc_access" {
  default = null
  type = object({
    # Currently, only one network interface is supported.
    network_interfaces = list(object({
      network    = string
      subnetwork = string
      tags       = optional(list(string))
    }))
    # Egress is one of "PRIVATE_RANGES_ONLY", "ALL_TRAFFIC", or "ALL_PRIVATE_RANGES"
    egress = string
  })
  description = "The VPC to send egress to. For more information, visit https://cloud.google.com/run/docs/configuring/vpc-direct-vpc"
}

variable "volumes" {
  description = "The volumes to make available to the container in the job for mounting."
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
  }))
  default = []
}

variable "volume_mounts" {
  description = "The volume mounts to mount the volumes to the container in the job."
  type = list(object({
    name       = string
    mount_path = string
  }))
  default = []
}

variable "notification_channels" {
  description = "List of notification channels to alert."
  type        = list(string)
}

variable "paused" {
  description = "Whether the cron scheduler is paused or not."
  type        = bool
  default     = false
}

variable "invokers" {
  description = "List of iam members invoker perimssions to invoke the job."
  type        = list(string)
  default     = []
}

variable "exec" {
  description = "Whether to execute job on modify."
  type        = bool
  default     = false
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
  description = "How long the absence of successful executions must persist before the alert fires. If not set or 0, defaults to success_alert_alignment_period_seconds for backward compatibility. This is the 'trigger absence time' in GCP monitoring terms."
  type        = number
  default     = 0
  validation {
    # GCP maximum trigger absence time is 23.5 hours (84600 seconds)
    condition = var.success_alert_duration_seconds == 0 || (
      var.success_alert_duration_seconds >= 60 &&
      var.success_alert_duration_seconds <= 84600
    )
    error_message = "Duration must be either 0 (to use alignment period value) or between 60 seconds and 23.5 hours (GCP maximum)."
  }
}

variable "enable_otel_sidecar" {
  description = "Enable otel sidecar for metrics"
  type        = bool
  default     = true
}

variable "otel_collector_image" {
  type        = string
  default     = "chainguard/opentelemetry-collector-contrib:latest"
  description = "The otel collector image to use as a base. Must be on gcr.io or dockerhub."
}

variable "scheduled_env_overrides" {
  type = list(object({
    name  = string
    value = string
  }))
  default     = []
  description = "List of env object overrides."
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

variable "team" {
  description = "Team label to apply to resources (replaces deprecated 'squad')."
  type        = string
}

variable "product" {
  description = "Product label to apply to the service."
  type        = string
  default     = "unknown"
}
