# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

mock_provider "google" {
  mock_resource "google_compute_managed_ssl_certificate" {
    defaults = {
      project = "provider-project"
    }
  }
}

variables {
  project_id   = "fixture-project"
  dns_zone     = "fixture-zone"
  enable_ipv6  = true
  ipv4_address = "192.0.2.10"
  ipv6_address = "2001:db8::10"
  service_hostnames = {
    "api.example.dev" = {
      name = "api"
    }
  }
  bucket_hostnames = {
    "downloads.example.dev" = {
      name = "downloads"
    }
  }
}

run "managed_dns_preserves_records_and_certificates" {
  command = apply

  assert {
    condition = jsonencode(concat(
      [for record in google_dns_record_set.public-service : {
        project = record.project
        zone    = record.managed_zone
        name    = record.name
        type    = record.type
        ttl     = record.ttl
        rrdatas = record.rrdatas
      }],
      [for record in google_dns_record_set.public-service-v6 : {
        project = record.project
        zone    = record.managed_zone
        name    = record.name
        type    = record.type
        ttl     = record.ttl
        rrdatas = record.rrdatas
      }],
      [for record in google_dns_record_set.bucket : {
        project = record.project
        zone    = record.managed_zone
        name    = record.name
        type    = record.type
        ttl     = record.ttl
        rrdatas = record.rrdatas
      }],
      [for record in google_dns_record_set.bucket-v6 : {
        project = record.project
        zone    = record.managed_zone
        name    = record.name
        type    = record.type
        ttl     = record.ttl
        rrdatas = record.rrdatas
      }],
      )) == jsonencode([
      {
        project = "fixture-project"
        zone    = "fixture-zone"
        name    = "api.example.dev."
        type    = "A"
        ttl     = 60
        rrdatas = ["192.0.2.10"]
      },
      {
        project = "fixture-project"
        zone    = "fixture-zone"
        name    = "api.example.dev."
        type    = "AAAA"
        ttl     = 60
        rrdatas = ["2001:db8::10"]
      },
      {
        project = "fixture-project"
        zone    = "fixture-zone"
        name    = "downloads.example.dev."
        type    = "A"
        ttl     = 60
        rrdatas = ["192.0.2.10"]
      },
      {
        project = "fixture-project"
        zone    = "fixture-zone"
        name    = "downloads.example.dev."
        type    = "AAAA"
        ttl     = 60
        rrdatas = ["2001:db8::10"]
      },
    ])
    error_message = "A configured DNS zone must create exactly the service and bucket address records."
  }

  assert {
    condition = jsonencode({
      services = { for host, cert in google_compute_managed_ssl_certificate.public-service : host => cert.managed[0].domains }
      buckets  = { for host, cert in google_compute_managed_ssl_certificate.bucket : host => cert.managed[0].domains }
      }) == jsonencode({
      services = { "api.example.dev" = ["api.example.dev."] }
      buckets  = { "downloads.example.dev" = ["downloads.example.dev."] }
    })
    error_message = "Managed certificates must preserve the configured hostnames."
  }
}

run "certificate_map_omits_managed_certificates_but_keeps_records" {
  command = apply

  variables {
    manage_certificates = false
  }

  assert {
    condition = jsonencode({
      services     = keys(google_compute_managed_ssl_certificate.public-service)
      buckets      = keys(google_compute_managed_ssl_certificate.bucket)
      certificates = output.ssl_certificates
      records      = output.records
      }) == jsonencode({
      services     = []
      buckets      = []
      certificates = null
      records      = run.managed_dns_preserves_records_and_certificates.records
    })
    error_message = "Certificate-map mode must retain DNS and omit per-hostname certificates."
  }
}

run "ipv4_only_has_no_aaaa_records" {
  command = apply

  variables {
    enable_ipv6      = false
    bucket_hostnames = {}
  }

  assert {
    condition = {
      services_v6 = keys(google_dns_record_set.public-service-v6)
      buckets_v6  = keys(google_dns_record_set.bucket-v6)
      } == {
      services_v6 = []
      buckets_v6  = []
    }
    error_message = "An IPv4-only frontend must not create AAAA records."
  }
}

run "disabled_hostnames_retain_dns_and_certificates" {
  command = apply

  variables {
    service_hostnames = {
      "api.example.dev" = {
        name = "api"
      }
      "disabled.example.dev" = {
        name     = "disabled"
        disabled = true
      }
    }
    bucket_hostnames = {
      "disabled-downloads.example.dev" = {
        name     = "disabled-downloads"
        disabled = true
      }
      "downloads.example.dev" = {
        name = "downloads"
      }
    }
  }

  assert {
    condition = jsonencode({
      records  = [for record in output.records : "${record.name}/${record.type}"]
      services = keys(google_compute_managed_ssl_certificate.public-service)
      buckets  = keys(google_compute_managed_ssl_certificate.bucket)
      attached = output.ssl_certificates
      }) == jsonencode({
      records = [
        "api.example.dev./A",
        "disabled.example.dev./A",
        "api.example.dev./AAAA",
        "disabled.example.dev./AAAA",
        "disabled-downloads.example.dev./A",
        "downloads.example.dev./A",
        "disabled-downloads.example.dev./AAAA",
        "downloads.example.dev./AAAA",
      ]
      services = ["api.example.dev", "disabled.example.dev"]
      buckets  = ["disabled-downloads.example.dev", "downloads.example.dev"]
      attached = [
        google_compute_managed_ssl_certificate.public-service["api.example.dev"].id,
        google_compute_managed_ssl_certificate.bucket["downloads.example.dev"].id,
      ]
    })
    error_message = "Disabled hostnames must retain DNS and certificates but must not attach certificates to the proxy."
  }
}

run "ipv6_requires_an_address" {
  command = plan

  variables {
    enable_ipv6  = true
    ipv6_address = null
  }

  expect_failures = [var.ipv6_address]
}

run "unset_certificate_project_uses_the_provider_project" {
  command = apply

  assert {
    condition = jsonencode({
      configured_project = var.certificate_project_id
      services = {
        for host, certificate in google_compute_managed_ssl_certificate.public-service : host => certificate.project
      }
      buckets = {
        for host, certificate in google_compute_managed_ssl_certificate.bucket : host => certificate.project
      }
      }) == jsonencode({
      configured_project = null
      services = {
        "api.example.dev" = "provider-project"
      }
      buckets = {
        "downloads.example.dev" = "provider-project"
      }
    })
    error_message = "An unset certificate project must preserve the provider project for service and bucket certificates."
  }
}

run "certificate_project_can_differ_from_the_dns_project" {
  command = apply

  variables {
    certificate_project_id = "certificates-project"
  }

  assert {
    condition = jsonencode({
      services = {
        for host, certificate in google_compute_managed_ssl_certificate.public-service : host => certificate.project
      }
      buckets = {
        for host, certificate in google_compute_managed_ssl_certificate.bucket : host => certificate.project
      }
      records = {
        for record in output.records : "${record.name}/${record.type}" => record.project
      }
      }) == jsonencode({
      services = {
        "api.example.dev" = "certificates-project"
      }
      buckets = {
        "downloads.example.dev" = "certificates-project"
      }
      records = {
        "api.example.dev./A"          = "fixture-project"
        "api.example.dev./AAAA"       = "fixture-project"
        "downloads.example.dev./A"    = "fixture-project"
        "downloads.example.dev./AAAA" = "fixture-project"
      }
    })
    error_message = "Service and bucket certificates must use certificate_project_id without changing the DNS project."
  }
}
