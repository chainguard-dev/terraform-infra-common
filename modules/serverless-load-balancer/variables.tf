# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

variable "name" {
  type = string
}

variable "project_id" {
  type = string
}

variable "ipv4_address" {
  description = "An existing global IPv4 address or its resource ID for the forwarding rule."
  type        = string
}

variable "ipv6_address" {
  description = "An existing global IPv6 address or its resource ID when enable_ipv6 is true."
  type        = string
  default     = null

  validation {
    condition     = !var.enable_ipv6 || var.ipv6_address != null
    error_message = "ipv6_address is required when enable_ipv6 is true."
  }
}

variable "enable_ipv6" {
  description = "Create an IPv6 forwarding rule. This flag must be known during planning."
  type        = bool
  default     = false
}

variable "ssl_certificates" {
  description = "Certificate resource IDs to attach to the HTTPS proxy. May remain attached alongside certificate_map during a migration."
  type        = list(string)
  default     = null
}

variable "certificate_map" {
  description = "Certificate Manager map reference for the HTTPS proxy, or an empty string when using ssl_certificates only."
  type        = string
  nullable    = false
  default     = ""
}

variable "regions" {
  description = "Regions containing serverless NEGs. Add regions here before adding them to serving_regions."
  type        = list(string)
  default     = ["us-central1"]
}

variable "serving_regions" {
  description = "Regions whose NEGs receive traffic. Remove regions here before removing them from regions."
  type        = list(string)
  default     = ["us-central1"]
}

variable "public-services" {
  description = "Hostnames mapped to existing regional Cloud Run services. Disabled services retain NEGs and backends but have no URL-map routes. CORS requires EXTERNAL_MANAGED on both the backend and forwarding rules."
  type = map(object({
    name                                          = string
    disabled                                      = optional(bool, false)
    external_managed_migration_state              = optional(string, null)
    external_managed_migration_testing_percentage = optional(number, null)
    load_balancing_scheme                         = optional(string, "EXTERNAL")
    connection_draining_timeout_sec               = optional(number, 300)
    cors_policy = optional(object({
      allow_origins        = optional(list(string), [])
      allow_origin_regexes = optional(list(string), [])
      allow_methods        = optional(list(string), ["GET", "HEAD", "POST", "OPTIONS"])
      allow_headers        = optional(list(string), ["Authorization", "Content-Type"])
      expose_headers       = optional(list(string), [])
      max_age              = optional(number, 3600)
      allow_credentials    = optional(bool, false)
    }), null)
  }))

  validation {
    condition = alltrue([
      for host, svc in var.public-services :
      svc.cors_policy == null || length(concat(svc.cors_policy.allow_origins, svc.cors_policy.allow_origin_regexes)) > 0
    ])
    error_message = "cors_policy must allow at least one origin, via allow_origins or allow_origin_regexes."
  }
}

variable "buckets" {
  description = "Hostnames mapped to existing Cloud Storage buckets and CDN settings. Disabled buckets have no backend or URL-map routes. Callers manage bucket access and signed-URL keys."
  type = map(object({
    name        = string
    bucket_name = string
    enable_cdn  = optional(bool, true)
    disabled    = optional(bool, false)
    cdn_policy = optional(object({
      cache_mode                   = optional(string)
      client_ttl                   = optional(number)
      default_ttl                  = optional(number)
      max_ttl                      = optional(number)
      signed_url_cache_max_age_sec = optional(number)
    }))
  }))
  default = {}
}

variable "iap" {
  description = "Backend-scoped IAP configuration for Cloud Run services. Callers manage IAP access and Cloud Run invocation grants."
  type = object({
    oauth2_client_id     = optional(string, null)
    oauth2_client_secret = optional(string, null)
    enabled              = bool
  })
  default = null
}

variable "security-policy" {
  description = "The security policy associated with service backends."
  type        = string
  default     = null
}

variable "forwarding_rule_load_balancing" {
  type = object({
    external_managed_backend_bucket_migration_state              = optional(string, null)
    external_managed_backend_bucket_migration_testing_percentage = optional(number, null)
    load_balancing_scheme                                        = optional(string, "EXTERNAL")
  })
  default = {}
}
