/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

output "bucket_name" {
  description = "The name of the status bucket, for wiring into the reconciler's status bucket env var (e.g. STATUS_BUCKET). Pair it with client.Bucket(name) and a gcsstatusmanager identity prefix."
  value       = google_storage_bucket.status.name
}

output "bucket_url" {
  description = "The gs:// URL of the status bucket."
  value       = google_storage_bucket.status.url
}
