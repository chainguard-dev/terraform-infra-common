/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

output "backend_buckets" {
  description = "The backend buckets created for buckets, keyed by hostname: name and id. A caller that attaches Cloud CDN signed-URL keys out of band attaches them to the name; the id is what the URL map routes the hostname's traffic through."
  value = {
    for host, bb in google_compute_backend_bucket.buckets : host => {
      name = bb.name
      id   = bb.id
    }
  }
}
