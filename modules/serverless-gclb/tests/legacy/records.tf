# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

output "records" {
  value = [for record in concat(
    values(google_dns_record_set.public-service),
    values(google_dns_record_set.public-service-v6),
    values(google_dns_record_set.bucket),
    values(google_dns_record_set.bucket-v6),
    ) : {
    project = record.project
    zone    = record.managed_zone
    name    = record.name
    type    = record.type
    ttl     = record.ttl
    rrdatas = record.rrdatas
  }]
}
