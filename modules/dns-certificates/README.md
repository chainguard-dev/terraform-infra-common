# DNS and certificates

This module creates A records, optional AAAA records and per-hostname
Google-managed SSL certificates in an existing DNS zone. It does not allocate
frontend addresses, create a zone or attach certificates to a proxy.

Use [`serverless-gclb`][assembly] as the convenience wrapper when one module
should allocate frontend addresses and configure DNS, certificates and load
balancing. For direct composition, [`serverless-load-balancer`][load-balancer]
attaches this module's certificate references to the HTTPS proxy.

`service_hostnames` and `bucket_hostnames` map hostnames to certificate resource
names. The separate maps preserve the resource instances used by
[`serverless-gclb`][assembly]. Disabled hostnames retain their records and
certificates but are excluded from the certificate references for a proxy.

The Google provider manages both DNS and certificates. A caller can select a
provider configuration for this component independently of the load-balancing
component. `certificate_project_id` defaults to the provider's project and can
differ from the DNS project. The provider's identity must already have the
required authority in both projects.

```hcl
module "domain" {
  source = "../dns-certificates"

  providers = {
    google = google.dns
  }

  project_id             = "dns-project"
  certificate_project_id = "workload-project"
  dns_zone               = "existing-zone"
  ipv4_address           = google_compute_global_address.endpoint.address

  service_hostnames = {
    "api.example.dev" = {
      name = "api"
    }
  }
}
```

Certificate resources can be created before issuance completes. Google activates
the certificates after DNS resolves to the frontend and the load balancer serves
the certificates. IPv6 uses an explicit `enable_ipv6` flag so the resource count
remains known when the address is allocated during the same apply.

For an existing certificate map, `manage_certificates = false` retains DNS and
omits per-hostname certificates. During a migration, keep certificate management
enabled while attaching the map, then disable it in a later apply. The
[assembly's migration instructions][migration] describe the two-step sequence.

The generated Outputs section describes the certificate references, managed
records and resource inventory. Consumers can test their composition against
these module-managed values.

[assembly]: ../serverless-gclb/
[load-balancer]: ../serverless-load-balancer/
[migration]: ../serverless-gclb/#migrating-an-existing-proxy-to-a-certificate-map

<!-- BEGIN_TF_DOCS -->
## Requirements

| Name | Version |
| ---- | ------- |
| <a name="requirement_terraform"></a> [terraform](#requirement\_terraform) | >= 1.9 |
| <a name="requirement_google"></a> [google](#requirement\_google) | >= 8.7.0 |

## Providers

| Name | Version |
| ---- | ------- |
| <a name="provider_google"></a> [google](#provider\_google) | >= 8.7.0 |

## Modules

No modules.

## Resources

| Name | Type |
| ---- | ---- |
| [google_compute_managed_ssl_certificate.bucket](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_managed_ssl_certificate) | resource |
| [google_compute_managed_ssl_certificate.public-service](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_managed_ssl_certificate) | resource |
| [google_dns_record_set.bucket](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/dns_record_set) | resource |
| [google_dns_record_set.bucket-v6](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/dns_record_set) | resource |
| [google_dns_record_set.public-service](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/dns_record_set) | resource |
| [google_dns_record_set.public-service-v6](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/dns_record_set) | resource |

## Inputs

| Name | Description | Type | Default | Required |
| ---- | ----------- | ---- | ------- | :------: |
| <a name="input_bucket_hostnames"></a> [bucket\_hostnames](#input\_bucket\_hostnames) | Bucket hostnames mapped to certificate resource names, with the same disabled behavior as service\_hostnames. | <pre>map(object({<br/>    name     = string<br/>    disabled = optional(bool, false)<br/>  }))</pre> | `{}` | no |
| <a name="input_certificate_project_id"></a> [certificate\_project\_id](#input\_certificate\_project\_id) | The project for managed certificates. Defaults to the Google provider's project. | `string` | `null` | no |
| <a name="input_dns_zone"></a> [dns\_zone](#input\_dns\_zone) | The existing managed DNS zone for the address records. | `string` | n/a | yes |
| <a name="input_enable_ipv6"></a> [enable\_ipv6](#input\_enable\_ipv6) | Create AAAA records. This flag must be known during planning. | `bool` | `false` | no |
| <a name="input_ipv4_address"></a> [ipv4\_address](#input\_ipv4\_address) | The frontend IPv4 address for A records. | `string` | n/a | yes |
| <a name="input_ipv6_address"></a> [ipv6\_address](#input\_ipv6\_address) | The frontend IPv6 address for AAAA records when enable\_ipv6 is true. | `string` | `null` | no |
| <a name="input_manage_certificates"></a> [manage\_certificates](#input\_manage\_certificates) | Create per-hostname managed certificates. Disable when a certificate map supplies TLS. Keep enabled during the first step of a certificate-map migration. | `bool` | `true` | no |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | The project containing the existing DNS zone. | `string` | n/a | yes |
| <a name="input_service_hostnames"></a> [service\_hostnames](#input\_service\_hostnames) | Service hostnames mapped to certificate resource names. Disabled hostnames retain records and certificates but are excluded from ssl\_certificates. | <pre>map(object({<br/>    name     = string<br/>    disabled = optional(bool, false)<br/>  }))</pre> | `{}` | no |

## Outputs

| Name | Description |
| ---- | ----------- |
| <a name="output_records"></a> [records](#output\_records) | Module-managed DNS configuration for consumer tests and inspection. Includes disabled hostnames. |
| <a name="output_resource_ids"></a> [resource\_ids](#output\_resource\_ids) | IDs of all records and certificates managed by this module. |
| <a name="output_ssl_certificates"></a> [ssl\_certificates](#output\_ssl\_certificates) | Certificate IDs for enabled hostnames, ordered by service hostname then bucket hostname. Null when certificate management is disabled. |
<!-- END_TF_DOCS -->
