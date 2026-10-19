// Copyright 2026 Chainguard, Inc.
// SPDX-License-Identifier: Apache-2.0

variable "groups" {
  description = "Lowercase emails of the Google groups to resolve, without a group: prefix (e.g. oncall@my-corp.dev)."
  type        = set(string)
  nullable    = false

  validation {
    condition     = alltrue([for g in var.groups : can(regex("^[^\\sA-Z\"\\\\:@]+@[^\\sA-Z\"\\\\:@]+$", g))])
    error_message = "groups entries must be bare lowercase emails, without a group: prefix (e.g. oncall@my-corp.dev)."
  }
}
