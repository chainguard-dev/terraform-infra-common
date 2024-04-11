/*
Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

variable "labels" {
  description = "Additional labels to apply to the dashboard."
  default     = {}
}

variable "alerts" {
  description = "A mapping from alerting policy names to the alert ids to add to the dashboard."
  type        = map(string)
  default     = {}
}

variable "project_id" {
  description = "ID of the GCP project"
  type        = string
}
