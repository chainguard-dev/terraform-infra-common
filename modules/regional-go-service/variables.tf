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

variable "ingress" {
  type        = string
  description = <<EOD
Which type of ingress traffic to accept for the service.

- INGRESS_TRAFFIC_ALL accepts all traffic, enabling the public .run.app URL for the service
- INGRESS_TRAFFIC_INTERNAL_LOAD_BALANCER accepts traffic only from a load balancer
- INGRESS_TRAFFIC_INTERNAL_ONLY accepts internal traffic only
EOD
  default     = "INGRESS_TRAFFIC_INTERNAL_ONLY"
}

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
  description = "The service account as which to run the service."
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
      base_image  = optional(string, "cgr.dev/chainguard/static:latest-glibc@sha256:092aad9f6448695b6e20333a8faa93fe3637bcf4e88aa804b8f01545eaf288bd")
      working_dir = string
      importpath  = string
      env         = optional(list(string), [])
    })
    args = optional(list(string), [])
    ports = optional(list(object({
      name           = optional(string, "http1")
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
    startup_probe = optional(object({
      initial_delay_seconds = optional(number)
      // GCP Terraform provider defaults differ from Cloud Run defaults.
      // See https://cloud.google.com/run/docs/configuring/healthchecks#tcp-startup-probe
      period_seconds    = optional(number, 240)
      timeout_seconds   = optional(number, 240)
      failure_threshold = optional(number, 1)
      http_get = optional(object({
        path = string
        port = optional(number)
      }), null)
      tcp_socket = optional(object({
        port = optional(number)
      }), null)
      grpc = optional(object({
        service = optional(string)
        port    = optional(number)
      }), null)
    }))
    liveness_probe = optional(object({
      initial_delay_seconds = optional(number)
      // GCP Terraform provider defaults differ from Cloud Run defaults.
      // See https://cloud.google.com/run/docs/configuring/healthchecks#tcp-startup-probe
      period_seconds    = optional(number, 240)
      timeout_seconds   = optional(number, 240)
      failure_threshold = optional(number, 1)
      http_get = optional(object({
        path = string
        port = optional(number)
      }), null)
      tcp_socket = optional(object({
        port = optional(number)
      }), null)
      grpc = optional(object({
        service = optional(string)
        port    = optional(number)
      }), null)
    }))
  }))
}

variable "scaling" {
  description = "The scaling configuration for the service."
  type = object({
    min_instances                    = optional(number, 0)
    max_instances                    = optional(number, 100)
    max_instance_request_concurrency = optional(number)
  })
  default = {}
}

variable "regional-volumes" {
  description = "The volumes to make available to the containers in the service for mounting."
  type = list(object({
    name = string
    gcs = optional(map(object({
      bucket    = string
      read_only = optional(bool, true)
    })), {})
    nfs = optional(map(object({
      server    = string
      path      = string
      read_only = optional(bool, true)
    })), {})
  }))
  default = []
}

variable "volumes" {
  description = "The volumes to make available to the containers in the service for mounting."
  type = list(object({
    name = string
    empty_dir = optional(object({
      medium     = optional(string, "MEMORY")
      size_limit = optional(string, "2G")
    }))
    secret = optional(object({
      secret = string
      items = list(object({
        version = string
        path    = string
      }))
    }))
  }))
  default = []
}

// https://cloud.google.com/run/docs/configuring/request-timeout
variable "request_timeout_seconds" {
  description = "The timeout for requests to the service, in seconds."
  type        = number
  default     = 300
}

variable "notification_channels" {
  description = "List of notification channels to alert."
  type        = list(string)
}

variable "execution_environment" {
  description = "The execution environment for the service"
  type        = string
  default     = "EXECUTION_ENVIRONMENT_GEN1"
  validation {
    error_message = "Must be EXECUTION_ENVIRONMENT_GEN1 or EXECUTION_ENVIRONMENT_GEN2. Got ${var.execution_environment}"
    condition     = var.execution_environment == "EXECUTION_ENVIRONMENT_GEN1" || var.execution_environment == "EXECUTION_ENVIRONMENT_GEN2"
  }
}

variable "labels" {
  description = "Labels to apply to the service."
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

variable "otel_collector_image" {
  type        = string
  default     = "chainguard/opentelemetry-collector-contrib:latest"
  description = "The otel collector image to use as a base. Must be on gcr.io or dockerhub."
}

variable "enable_profiler" {
  type        = bool
  default     = false
  description = "Enable cloud profiler."
}

variable "otel_resources" {
  type = object({
    limits = optional(object(
      {
        cpu    = string
        memory = string
      }
    ), null)
    cpu_idle          = optional(bool)
    startup_cpu_boost = optional(bool)
  })
  default     = null
  description = "The resource clause for otel sidecar container."
}

variable "product" {
  description = "Product label to apply to the service."
  type        = string
  default     = "unknown"
}
