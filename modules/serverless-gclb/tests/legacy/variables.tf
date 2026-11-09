# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

variable "project_id" {
  type = string
}

variable "name" {
  type = string
}

variable "dns_zone" {
  type = string
}

variable "regions" {
  type    = list(string)
  default = ["us-central1"]
}

variable "serving_regions" {
  type    = list(string)
  default = ["us-central1"]
}

variable "public-services" {
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
}

variable "buckets" {
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
  type = object({
    oauth2_client_id     = optional(string, null)
    oauth2_client_secret = optional(string, null)
    enabled              = bool
  })
  default = null
}

variable "security-policy" {
  type    = string
  default = null
}

variable "forwarding_rule_load_balancing" {
  type = object({
    external_managed_backend_bucket_migration_state              = optional(string, null)
    external_managed_backend_bucket_migration_testing_percentage = optional(number, null)
    load_balancing_scheme                                        = optional(string, "EXTERNAL")
  })
  default = {}
}

variable "enable_ipv6" {
  type    = bool
  default = false
}

variable "certificate_map" {
  type    = string
  default = ""
}

variable "retain_managed_certificates" {
  type    = bool
  default = false
}
