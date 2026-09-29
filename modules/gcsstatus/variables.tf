/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

variable "project_id" {
  description = "The GCP project ID where the status bucket will be created."
  type        = string
}

variable "name" {
  description = "Base name for the status bucket. A short random suffix is appended to keep the (globally unique) bucket name collision-free."
  type        = string
}

variable "location" {
  description = "The location (region or multi-region) for the status bucket."
  type        = string
}

variable "writer_service_accounts" {
  description = "Service account members (e.g. serviceAccount:foo@project.iam.gserviceaccount.com) granted read+write on the status bucket. gcsstatusmanager overwrites objects, so roles/storage.objectUser (no repoAdmin/delete privilege needed for writes) is granted."
  type        = list(string)
  default     = []
}

variable "reader_service_accounts" {
  description = "Service account members granted read-only (roles/storage.objectViewer) access, for consumers built with gcsstatusmanager.NewReadOnly."
  type        = list(string)
  default     = []
}

variable "lifecycle_age_days" {
  description = "When > 0, adds a bucket lifecycle rule that deletes status objects older than this many days. Status objects are cheap and self-heal, so a TTL bounds the cost of abandoned entries. 0 disables the rule."
  type        = number
  default     = 0
}

variable "resource_manager_tags" {
  description = "Resource Manager tags to bind to the status bucket, as tagKeys/<id> => tagValues/<id>."
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
