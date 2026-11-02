/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

variable "project_id" {
  description = "The project the key ring is created in."
  type        = string
}

variable "key_ring_name" {
  description = "The name of the key ring, in the global location. Cloud KMS never deletes a key ring, so a new name leaves the old ring and its key behind."
  type        = string
}

variable "key_name" {
  description = "The name of the import-only key that holds the App's private key."
  type        = string
}

variable "signer_service_account" {
  description = "The email of the service account that signs the App's JWTs with the key."
  type        = string
}

variable "destroy_scheduled_duration" {
  description = "How long a destroyed key version stays recoverable before Cloud KMS destroys it, such as \"86400s\". Null takes the Cloud KMS default of 30 days. Changing it replaces the key, so set it only on a new key."
  type        = string
  default     = null
}
