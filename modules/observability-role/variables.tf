/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

variable "project_id" {
  description = "The project in which to create the custom role."
  type        = string
}

variable "role_id" {
  description = "The role_id of the custom role."
  type        = string
  default     = "serviceObservability"
}

variable "title" {
  description = "Human-readable title of the custom role."
  type        = string
  default     = "Service Observability"
  // The limit is 100 UTF-8 bytes (https://cloud.google.com/iam/quotas), and
  // length() counts characters, so count the bytes that base64 encodes.
  validation {
    condition     = length(base64encode(var.title)) / 4 * 3 - length(regexall("=", base64encode(var.title))) <= 100
    error_message = "title must be at most 100 bytes of UTF-8, the custom role title limit, but '${var.title}' is ${length(base64encode(var.title)) / 4 * 3 - length(regexall("=", base64encode(var.title)))} bytes."
  }
}
