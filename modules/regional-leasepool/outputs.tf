# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

output "buckets" {
  description = "Lease bucket names keyed by region, ready for regional environment configuration."
  value       = { for region, bucket in google_storage_bucket.leases : region => bucket.name }
  depends_on  = [google_storage_bucket_iam_member.participant]
}
