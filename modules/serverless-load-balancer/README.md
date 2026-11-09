# Serverless load balancer

This module creates regional serverless NEGs, service and bucket backends,
hostname routing, an HTTPS proxy, a TLS policy and global forwarding rules. The
caller supplies existing global frontend addresses and certificate references.
The module does not create Cloud Run services, storage buckets, addresses, DNS
records or certificates.

Use [`serverless-gclb`][assembly] as the convenience wrapper when one module
should allocate frontend addresses and configure DNS, certificates and load
balancing. For direct composition, [`dns-certificates`][dns-certificates]
creates DNS records and managed certificates for existing frontend addresses.

The `public-services` map connects hostname keys to existing regional Cloud Run
services through `name`. The `buckets` map connects hostname keys to existing
Cloud Storage buckets through `bucket_name`. These interfaces and their defaults
match [`serverless-gclb`][assembly], including CORS, serving-region selection,
connection draining, CDN settings for buckets and load-balancer migration fields.

```hcl
module "load_balancer" {
  source = "../serverless-load-balancer"

  project_id       = "workload-project"
  name             = "api"
  ipv4_address     = google_compute_global_address.endpoint.id
  ssl_certificates = module.domain.ssl_certificates

  public-services = {
    "api.example.dev" = {
      name                  = "api"
      load_balancing_scheme = "EXTERNAL_MANAGED"
    }
  }

  forwarding_rule_load_balancing = {
    load_balancing_scheme = "EXTERNAL_MANAGED"
  }

  iap = {
    enabled = true
  }
}
```

IAP applies to service backends. Callers manage backend-scoped access grants and
Cloud Run invocation grants. Service backends have CDN disabled; bucket backends
keep their independent CDN configuration and access requirements. Disabled
services retain NEGs and backends but have no routes. Disabled buckets have
neither backends nor routes.

`certificate_map` and `ssl_certificates` can both be attached during a
certificate-map migration. The [assembly's migration instructions][migration]
describe the required two-step sequence. IPv6 uses an explicit `enable_ipv6` flag
so the forwarding-rule count remains known during planning.

The generated Outputs section describes backend identifiers and the resource
inventory. Consumers can attach their own IAM grants and test their composition
against module-managed resources. These outputs do not describe inherited or
out-of-band authority.

[assembly]: ../serverless-gclb/
[dns-certificates]: ../dns-certificates/
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
| [google_compute_backend_bucket.buckets](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_backend_bucket) | resource |
| [google_compute_backend_service.public-services](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_backend_service) | resource |
| [google_compute_global_forwarding_rule.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_global_forwarding_rule) | resource |
| [google_compute_global_forwarding_rule.this-v6](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_global_forwarding_rule) | resource |
| [google_compute_region_network_endpoint_group.regional-backends](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_region_network_endpoint_group) | resource |
| [google_compute_ssl_policy.ssl_policy](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_ssl_policy) | resource |
| [google_compute_target_https_proxy.public-service](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_target_https_proxy) | resource |
| [google_compute_url_map.public-service](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_url_map) | resource |
| [google_storage_bucket.buckets](https://registry.terraform.io/providers/hashicorp/google/latest/docs/data-sources/storage_bucket) | data source |

## Inputs

| Name | Description | Type | Default | Required |
| ---- | ----------- | ---- | ------- | :------: |
| <a name="input_buckets"></a> [buckets](#input\_buckets) | Hostnames mapped to existing Cloud Storage buckets and CDN settings. Disabled buckets have no backend or URL-map routes. Callers manage bucket access and signed-URL keys. | <pre>map(object({<br/>    name        = string<br/>    bucket_name = string<br/>    enable_cdn  = optional(bool, true)<br/>    disabled    = optional(bool, false)<br/>    cdn_policy = optional(object({<br/>      cache_mode                   = optional(string)<br/>      client_ttl                   = optional(number)<br/>      default_ttl                  = optional(number)<br/>      max_ttl                      = optional(number)<br/>      signed_url_cache_max_age_sec = optional(number)<br/>    }))<br/>  }))</pre> | `{}` | no |
| <a name="input_certificate_map"></a> [certificate\_map](#input\_certificate\_map) | Certificate Manager map reference for the HTTPS proxy, or an empty string when using ssl\_certificates only. | `string` | `""` | no |
| <a name="input_enable_ipv6"></a> [enable\_ipv6](#input\_enable\_ipv6) | Create an IPv6 forwarding rule. This flag must be known during planning. | `bool` | `false` | no |
| <a name="input_forwarding_rule_load_balancing"></a> [forwarding\_rule\_load\_balancing](#input\_forwarding\_rule\_load\_balancing) | n/a | <pre>object({<br/>    external_managed_backend_bucket_migration_state              = optional(string, null)<br/>    external_managed_backend_bucket_migration_testing_percentage = optional(number, null)<br/>    load_balancing_scheme                                        = optional(string, "EXTERNAL")<br/>  })</pre> | `{}` | no |
| <a name="input_iap"></a> [iap](#input\_iap) | Backend-scoped IAP configuration for Cloud Run services. Callers manage IAP access and Cloud Run invocation grants. | <pre>object({<br/>    oauth2_client_id     = optional(string, null)<br/>    oauth2_client_secret = optional(string, null)<br/>    enabled              = bool<br/>  })</pre> | `null` | no |
| <a name="input_ipv4_address"></a> [ipv4\_address](#input\_ipv4\_address) | An existing global IPv4 address or its resource ID for the forwarding rule. | `string` | n/a | yes |
| <a name="input_ipv6_address"></a> [ipv6\_address](#input\_ipv6\_address) | An existing global IPv6 address or its resource ID when enable\_ipv6 is true. | `string` | `null` | no |
| <a name="input_name"></a> [name](#input\_name) | n/a | `string` | n/a | yes |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | n/a | `string` | n/a | yes |
| <a name="input_public-services"></a> [public-services](#input\_public-services) | Hostnames mapped to existing regional Cloud Run services. Disabled services retain NEGs and backends but have no URL-map routes. CORS requires EXTERNAL\_MANAGED on both the backend and forwarding rules. | <pre>map(object({<br/>    name                                          = string<br/>    disabled                                      = optional(bool, false)<br/>    external_managed_migration_state              = optional(string, null)<br/>    external_managed_migration_testing_percentage = optional(number, null)<br/>    load_balancing_scheme                         = optional(string, "EXTERNAL")<br/>    connection_draining_timeout_sec               = optional(number, 300)<br/>    cors_policy = optional(object({<br/>      allow_origins        = optional(list(string), [])<br/>      allow_origin_regexes = optional(list(string), [])<br/>      allow_methods        = optional(list(string), ["GET", "HEAD", "POST", "OPTIONS"])<br/>      allow_headers        = optional(list(string), ["Authorization", "Content-Type"])<br/>      expose_headers       = optional(list(string), [])<br/>      max_age              = optional(number, 3600)<br/>      allow_credentials    = optional(bool, false)<br/>    }), null)<br/>  }))</pre> | n/a | yes |
| <a name="input_regions"></a> [regions](#input\_regions) | Regions containing serverless NEGs. Add regions here before adding them to serving\_regions. | `list(string)` | <pre>[<br/>  "us-central1"<br/>]</pre> | no |
| <a name="input_security-policy"></a> [security-policy](#input\_security-policy) | The security policy associated with service backends. | `string` | `null` | no |
| <a name="input_serving_regions"></a> [serving\_regions](#input\_serving\_regions) | Regions whose NEGs receive traffic. Remove regions here before removing them from regions. | `list(string)` | <pre>[<br/>  "us-central1"<br/>]</pre> | no |
| <a name="input_ssl_certificates"></a> [ssl\_certificates](#input\_ssl\_certificates) | Certificate resource IDs to attach to the HTTPS proxy. May remain attached alongside certificate\_map during a migration. | `list(string)` | `null` | no |

## Outputs

| Name | Description |
| ---- | ----------- |
| <a name="output_backend_buckets"></a> [backend\_buckets](#output\_backend\_buckets) | Module-managed backend buckets by hostname. Callers can attach signed-URL keys using the backend name. |
| <a name="output_backend_services"></a> [backend\_services](#output\_backend\_services) | Module-managed service backends by hostname, for backend-scoped IAM and consumer tests. Excludes inherited and out-of-band access. |
| <a name="output_resource_ids"></a> [resource\_ids](#output\_resource\_ids) | IDs of all NEGs, backends, URL map, TLS policy, HTTPS proxy and forwarding rules managed by this module. |
<!-- END_TF_DOCS -->
