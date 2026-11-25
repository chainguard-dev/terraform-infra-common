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
  nullable    = false

  // An empty, zero or negative age would let the untagged delete policy match
  // every attestation bundle in the repository. The provider converts the
  // value to seconds in a 64-bit integer without an overflow check, so the
  // digit cap keeps a large value from wrapping negative.
  validation {
    condition     = can(regex("^[1-9][0-9]{0,9}[smhd]$", var.cleanup_policy_older_than))
    error_message = "cleanup_policy_older_than must be a positive whole number of at most 10 digits followed by s, m, h, or d, such as 86400s or 1d."
  }
}

variable "resource_manager_tags" {
  description = "Resource Manager tags to bind to the attestations repository, as tagKeys/<id> => tagValues/<id>."
  type        = map(string)
  default     = {}
  nullable    = false

  validation {
    condition = length(var.resource_manager_tags) <= 50 && alltrue([
      for key, value in var.resource_manager_tags :
      can(regex("^tagKeys/[0-9]+$", key)) && can(regex("^tagValues/[0-9]+$", value))
    ])
    error_message = "resource_manager_tags allows at most 50 entries; keys must be tagKeys/<numeric-id> and values tagValues/<numeric-id>."
  }
}
