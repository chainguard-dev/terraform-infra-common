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

variable "custom_audiences" {
  type        = list(string)
  description = <<EOD
Optional list of custom audiences accepted by the Cloud Run service's ID-token
validation, in addition to the service's default *.run.app URL. Required for
services reached by a non-run.app hostname or IP (e.g. behind an internal HTTP
ALB / Private Service Connect) where callers cannot use the run.app URL as the
token audience. Empty leaves custom audiences unset.
EOD
  default     = []
}

variable "egress" {
  type        = string
  description = <<EOD
Which type of egress traffic to send through the VPC.

- ALL_TRAFFIC sends all traffic through regional VPC network
- PRIVATE_RANGES_ONLY sends only traffic to private IP addresses through regional VPC network
EOD
  default     = "ALL_TRAFFIC"
}

variable "regional-egress" {
  type        = map(string)
  description = <<EOD
Optional per-region override of var.egress, keyed by region name. A region
present in this map uses its value; regions absent fall back to var.egress.
Use this to route a single region's public egress through the VPC (e.g. for a
stable Cloud NAT egress IP) without changing the others. Values must be one of
ALL_TRAFFIC or PRIVATE_RANGES_ONLY.
EOD
  default     = {}
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
    image   = string
    command = optional(list(string), [])
    args    = optional(list(string), [])
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
    cloud_sql_instance = optional(object({
      instances = list(string)
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
  default     = "EXECUTION_ENVIRONMENT_GEN2"
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

variable "team" {
  description = "Team label to apply to resources (replaces deprecated 'squad')."
  type        = string
}

variable "enable_otel_sidecar" {
  description = "Enable otel sidecar for metrics. Enabled by default, should only be disabled for exceptional cases."
  type        = bool
  default     = true
}

variable "otel_collector_image" {
  type        = string
  default     = "chainguard/opentelemetry-collector-contrib:latest"
  description = "The otel collector image to use as a base. Must be on gcr.io or dockerhub. The bundled scrape config enables native histogram scraping by default, which needs opentelemetry-collector-contrib v0.142.0 or later; older collectors reject the config at startup."
}

variable "scrape_native_histograms" {
  type        = bool
  default     = true
  description = "Scrape native (exponential) histograms from metrics targets. Requires opentelemetry-collector-contrib v0.142.0 or later. Set to false when pinning otel_collector_image to an older collector, which rejects the scrape keys at startup."
}

variable "enable_profiler" {
  type        = bool
  default     = false
  description = "Enable cloud profiler."
}

variable "observability_role" {
  type        = string
  default     = null
  description = "Fully-qualified id of a single role (e.g. from the observability-role module) to grant the service account in place of the three built-in observability roles (monitoring.metricWriter, cloudtrace.agent, cloudprofiler.agent). Collapsing to one role keeps large projects under the 1,500-member IAM policy limit."

  validation {
    condition     = var.observability_role == null || can(regex("^projects/[^/]+/roles/[^/]+$", var.observability_role))
    error_message = "observability_role must be a fully-qualified project role id: projects/{project}/roles/{role_id}."
  }
}

variable "enable_observability_iam" {
  type        = bool
  default     = true
  description = "Whether this module grants the service account the observability roles (monitoring.metricWriter, cloudtrace.agent, cloudprofiler.agent) on the project. Set false when the caller manages these grants itself, e.g. a service account shared across multiple services, where per-service grants would create overlapping non-authoritative IAM members that revoke each other on destroy."
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

variable "launch_stage" {
  description = "The launch stage of the Cloud Run service (e.g. BETA to leverage features like disk volumes)."
  type        = string
  default     = "GA"
}

variable "require_authenticated_invocations" {
  description = <<EOD
When true, do not grant `roles/run.invoker` to `allUsers` even when ingress
allows non-internal traffic. Use this for services that are publicly reachable
but should be gated by Cloud Run IAM (e.g. an admin dashboard accessed via
`gcloud run services proxy` and restricted to an engineering group). The
caller is responsible for granting `roles/run.invoker` to the appropriate
principals.

Default `false` preserves the existing behavior, where non-internal ingress
implies a load balancer handles authentication and the service is publicly
invokable.
EOD
  type        = bool
  default     = false
}
