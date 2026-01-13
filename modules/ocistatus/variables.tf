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
  description = "The service account member (e.g. serviceAccount:foo@project.iam.gserviceaccount.com) to grant write access."
  type        = string
}

variable "cleanup_policy_older_than" {
  description = "Duration after which untagged images are deleted (e.g. 86400s for 1 day)."
  type        = string
  default     = "86400s"
}
