/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

terraform {
  required_version = ">= 1.9"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = ">= 8.7.0"
    }
  }
}

resource "google_kms_key_ring" "this" {
  project  = var.project_id
  name     = var.key_ring_name
  location = "global"
}

# Holds the GitHub App's private key. KMS never generates a version of an
# import-only key, so the key imported with the import-github-app-key script
# beside this module is the only one it signs with.
resource "google_kms_crypto_key" "this" {
  name     = var.key_name
  key_ring = google_kms_key_ring.this.id
  purpose  = "ASYMMETRIC_SIGN"

  version_template {
    algorithm = "RSA_SIGN_PKCS1_2048_SHA256"
  }

  import_only                   = true
  skip_initial_version_creation = true
  destroy_scheduled_duration    = var.destroy_scheduled_duration

  # Destroying the key schedules the imported App key for destruction, after
  # which the App needs a new key generated in GitHub and imported. A
  # deliberate teardown removes this block in the same change.
  lifecycle {
    prevent_destroy = true
  }
}

resource "google_kms_crypto_key_iam_member" "signer" {
  crypto_key_id = google_kms_crypto_key.this.id
  role          = "roles/cloudkms.signer"
  member        = "serviceAccount:${var.signer_service_account}"
}
