# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

variable "project_id" {
  description = "Project containing the regional lease buckets."
  type        = string
}

variable "name" {
  description = "Bucket name prefix, at most 24 lowercase letters, digits or hyphens."
  type        = string
  validation {
    condition     = can(regex("^[a-z][a-z0-9-]{0,23}$", var.name))
    error_message = "name must begin with a letter and contain at most 24 lowercase letters, digits or hyphens."
  }
}

variable "regions" {
  description = "Region keys; accepts the same region map as regional-go-service and regional-go-cron. Each region has an independent lease namespace."
  type        = map(object({}))
  validation {
    condition     = length(var.regions) > 0 && alltrue([for r in keys(var.regions) : can(regex("^[a-z]+-[a-z]+[0-9]+$", r)) && length(r) <= 28])
    error_message = "regions must contain at least one GCP region name, each at most 28 characters."
  }
}

variable "identity" {
  description = "The single IAM member granted access, e.g. serviceAccount:worker@example-project.iam.gserviceaccount.com."
  type        = string
  validation {
    condition     = can(regex("^(serviceAccount:|user:|principal://).+$", var.identity))
    error_message = "identity must name a single service account, user or workload identity principal."
  }
}

variable "team" {
  description = "Owning team."
  type        = string
}

variable "product" {
  description = "Product label."
  type        = string
  default     = "unknown"
}

variable "labels" {
  description = "Additional bucket labels."
  type        = map(string)
  default     = {}
}

variable "force_destroy" {
  description = "Allow destruction of nonempty lease buckets. Stop all participants before destroying or replacing a bucket."
  type        = bool
  default     = false
}
