/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

output "backend_buckets" {
  description = "The backend buckets created for buckets, keyed by hostname: name and id. A caller serving a bucket to signed URLs only attaches its Cloud CDN signed-URL keys to the name, in Terraform or out of band so the key values never enter state, and grants Cloud CDN's fill service agent read on the storage bucket; the id is what the URL map routes the hostname's traffic through."
  value = {
    for host, bb in google_compute_backend_bucket.buckets : host => {
      name = bb.name
      id   = bb.id
    }
  }
}
