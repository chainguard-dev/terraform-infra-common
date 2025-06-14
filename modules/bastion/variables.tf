/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

variable "name" {
  description = "Name prefix for all resources (also used as network tag)."
  type        = string
}

variable "project_id" {
  description = "Project in which to deploy the bastion host."
  type        = string
}

variable "zone" {
  description = "Compute Engine zone for the bastion VM (e.g. us-central1-a)."
  type        = string
}

variable "network" {
  description = "VPC network self-link or name the bastion joins."
  type        = string
}

variable "subnetwork" {
  description = "Subnetwork name the bastion joins (must be private)."
  type        = string
}

variable "dev_principals" {
  description = "IAM principals (users, groups, or service accounts) granted OS Login & Cloud SQL access."
  type        = list(string)
}

variable "squad" {
  description = "Squad or team label applied to the instance (required)."
  type        = string

  validation {
    condition     = length(trim(var.squad, " \t\n\r")) > 0
    error_message = "squad must be specified and non-empty."
  }
}

variable "machine_type" {
  description = "Compute Engine machine type for the bastion."
  type        = string
  default     = "e2-micro"
}

variable "extra_sa_roles" {
  description = "Additional IAM roles to bind to the bastion's service account."
  type        = list(string)
  default     = []
}

variable "enable_nat" {
  description = "Whether to create a dedicated Cloud NAT router for outbound egress. Disable when VPC already has NAT."
  type        = bool
  default     = true
}

variable "patch_day" {
  description = "Day of week (in UTC) when OS Config patching runs."
  type        = string
  default     = "MONDAY"
  validation {
    condition     = contains(["MONDAY", "TUESDAY", "WEDNESDAY", "THURSDAY", "FRIDAY", "SATURDAY", "SUNDAY"], upper(var.patch_day))
    error_message = "patch_day must be a valid weekday name."
  }
}

variable "patch_time_utc" {
  description = "Time of day in HH:MM (UTC) when patching runs."
  type        = string
  default     = "03:00"
  validation {
    condition     = can(regex("^([01][0-9]|2[0-3]):[0-5][0-9]$", var.patch_time_utc))
    error_message = "patch_time_utc must be in HH:MM 24-hour format."
  }
}

variable "deletion_protection" {
  description = "GCE API deletion protection flag. When true, prevents instance deletion via the API."
  type        = bool
  default     = true
}

variable "install_sql_proxy" {
  description = "Whether to install the Cloud SQL Auth Proxy binary and grant associated IAM permissions."
  type        = bool
  default     = false
}
