# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

# The secret and placeholder addresses as the module declared them before
# the placeholder moved to a write-only version.

terraform {
  required_version = ">= 1.9"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = ">= 8.7.0"
    }
  }
}

variable "name" {
  type = string
}

resource "google_secret_manager_secret" "this" {
  secret_id = var.name
  replication {
    auto {}
  }
}

resource "google_secret_manager_secret_version" "placeholder" {
  count       = 1
  secret      = google_secret_manager_secret.this.id
  secret_data = "placeholder"
}
