/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

variable "project_id" {
  description = "The GCP project ID where the repository will be created."
  type        = string
}

variable "name" {
  description = "The name for the Artifact Registry repository."
  type        = string
}

variable "location" {
  description = "The location (region) for the Artifact Registry repository."
  type        = string
}

variable "service_account" {
  description = "The service account member (e.g. serviceAccount:foo@project.iam.gserviceaccount.com) to grant access to write and replace attestations."
  type        = string
}

variable "cleanup_policy_older_than" {
  description = "Duration after which untagged images are deleted (e.g. 86400s for 1 day)."
  type        = string
  default     = "86400s"
}

variable "resource_manager_tags" {
  description = "Resource Manager tags to bind to the attestations repository, as tagKeys/<id> => tagValues/<id>."
  type        = map(string)
  default     = {}
  nullable    = false

  validation {
    condition = alltrue([
      for key, value in var.resource_manager_tags :
      can(regex("^tagKeys/[0-9]+$", key)) && can(regex("^tagValues/[0-9]+$", value))
    ])
    error_message = "resource_manager_tags keys must be tagKeys/<numeric-id> and values must be tagValues/<numeric-id>."
  }
}
