# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

output "ssl_certificates" {
  description = "Certificate IDs for enabled hostnames, ordered by service hostname then bucket hostname. Null when certificate management is disabled."
  value = var.manage_certificates ? concat(
    [for host, cert in google_compute_managed_ssl_certificate.public-service : cert.id if !var.service_hostnames[host].disabled],
    [for host, cert in google_compute_managed_ssl_certificate.bucket : cert.id if !var.bucket_hostnames[host].disabled],
  ) : null
}

output "records" {
  description = "Module-managed DNS configuration for consumer tests and inspection. Includes disabled hostnames."
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

output "resource_ids" {
  description = "IDs of all records and certificates managed by this module."
  value = concat(
    [for record in google_dns_record_set.public-service : record.id],
    [for record in google_dns_record_set.public-service-v6 : record.id],
    [for record in google_dns_record_set.bucket : record.id],
    [for record in google_dns_record_set.bucket-v6 : record.id],
    [for cert in google_compute_managed_ssl_certificate.public-service : cert.id],
    [for cert in google_compute_managed_ssl_certificate.bucket : cert.id],
  )
}
