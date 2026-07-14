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
}
