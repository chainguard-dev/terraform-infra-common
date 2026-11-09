# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

variable "project_id" {
  description = "The project containing the existing DNS zone."
  type        = string
}

variable "certificate_project_id" {
  description = "The project for managed certificates. Defaults to the Google provider's project."
  type        = string
  default     = null
}

variable "dns_zone" {
  description = "The existing managed DNS zone for the address records."
  type        = string
  nullable    = false
}

variable "ipv4_address" {
  description = "The frontend IPv4 address for A records."
  type        = string
}

variable "ipv6_address" {
  description = "The frontend IPv6 address for AAAA records when enable_ipv6 is true."
  type        = string
  default     = null

  validation {
    condition     = !var.enable_ipv6 || var.ipv6_address != null
    error_message = "ipv6_address is required when enable_ipv6 is true."
  }
}

variable "enable_ipv6" {
  description = "Create AAAA records. This flag must be known during planning."
  type        = bool
  default     = false
}

variable "service_hostnames" {
  description = "Service hostnames mapped to certificate resource names. Disabled hostnames retain records and certificates but are excluded from ssl_certificates."
  type = map(object({
    name     = string
    disabled = optional(bool, false)
  }))
  default = {}
}

variable "bucket_hostnames" {
  description = "Bucket hostnames mapped to certificate resource names, with the same disabled behavior as service_hostnames."
  type = map(object({
    name     = string
    disabled = optional(bool, false)
  }))
  default = {}
}

variable "manage_certificates" {
  description = "Create per-hostname managed certificates. Disable when a certificate map supplies TLS. Keep enabled during the first step of a certificate-map migration."
  type        = bool
  nullable    = false
  default     = true
}
