/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

output "key_id" {
  description = "The resource name of the key, projects/<project>/locations/global/keyRings/<ring>/cryptoKeys/<key>, which the import-github-app-key script takes. A gcpkms:// key URI appends /cryptoKeyVersions/<N>."
  value       = google_kms_crypto_key.this.id
}
