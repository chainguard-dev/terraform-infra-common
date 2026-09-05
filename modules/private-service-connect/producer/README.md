# `private-service-connect/producer`

This submodule fronts an existing **regional internal Cloud Run service** with a
**regional internal Application Load Balancer** (INTERNAL_MANAGED) and publishes
it via a **Private Service Connect (PSC) service attachment**.

The resource graph is:

```
Cloud Run service
  -> serverless NEG
  -> regional internal backend service (INTERNAL_MANAGED, HTTP)
  -> regional URL map
  -> regional target HTTP proxy
  -> internal ALB forwarding rule (the LB VIP)
  -> PSC service attachment (ACCEPT_MANUAL)
```

The caller is responsible for creating the `REGIONAL_MANAGED_PROXY` proxy-only
subnet and the `PRIVATE_SERVICE_CONNECT` NAT subnet(s); this module only accepts
their self-links. The `proxy_only_subnet` self-link is used purely to order the
ALB forwarding rule after the proxy-only subnet exists (via `depends_on`).

HTTP (not HTTPS) is intentional on the internal ALB: TLS to the underlying
`run.app` backend is terminated by the serverless NEG, and inbound authorization
is enforced via Cloud Run invoker IAM (configured separately, not by this
module).

Hand the `service_attachment_id` output to the `consumer` submodule (typically
across Terraform states via a tfvar). See the parent module's
[README](../README.md) for the producer -> consumer flow and the two-phase apply.

Changing `allow_global_access`, or switching the frontend between HTTP and HTTPS
(`tls_certificates`), recreates the forwarding rule and, by dependency, the
service attachment: GCP does not allow an in-place change to those fields while
the forwarding rule is referenced by a PSC service attachment. Deleting the
attachment **closes every connected consumer endpoint for good**: a `CLOSED`
PSC connection is terminal, the endpoint does not reconnect to the recreated
attachment (`ACCEPT_MANUAL` re-acceptance applies to new connections only), and
the consumer's forwarding rule shows no Terraform drift because the attachment's
self-link is unchanged. Treat such a flip as a coordinated change: after it
applies, every accepted consumer recreates its endpoint (the `consumer`
submodule's `connection_generation` input), and the seam is down for each
consumer until it does. See
[Private Service Connect endpoint status](https://cloud.google.com/vpc/docs/about-accessing-vpc-hosted-services-endpoints).

<!-- BEGIN_TF_DOCS -->
## Requirements

No requirements.

## Providers

| Name | Version |
|------|---------|
| <a name="provider_google"></a> [google](#provider\_google) | n/a |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [google_compute_forwarding_rule.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_forwarding_rule) | resource |
| [google_compute_region_backend_service.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_region_backend_service) | resource |
| [google_compute_region_network_endpoint_group.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_region_network_endpoint_group) | resource |
| [google_compute_region_target_http_proxy.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_region_target_http_proxy) | resource |
| [google_compute_region_target_https_proxy.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_region_target_https_proxy) | resource |
| [google_compute_region_url_map.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_region_url_map) | resource |
| [google_compute_service_attachment.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_service_attachment) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_allow_global_access"></a> [allow\_global\_access](#input\_allow\_global\_access) | Allow clients in any region to reach the internal ALB forwarding rule. Required before a PSC consumer endpoint targeting this service attachment can set allow\_psc\_global\_access = true; without it the consumer apply fails with "the producer service does not support consumer global access". Leave false when all consumers are in the producer's region. | `bool` | `false` | no |
| <a name="input_cloud_run_service_name"></a> [cloud\_run\_service\_name](#input\_cloud\_run\_service\_name) | Name of the existing regional internal Cloud Run service to front with the internal ALB. | `string` | n/a | yes |
| <a name="input_connection_limit"></a> [connection\_limit](#input\_connection\_limit) | Per-consumer connection limit applied to each entry in consumer\_accept\_projects. | `number` | `10` | no |
| <a name="input_consumer_accept_projects"></a> [consumer\_accept\_projects](#input\_consumer\_accept\_projects) | List of consumer project IDs or numbers explicitly accepted by the service attachment (ACCEPT\_MANUAL). | `list(string)` | n/a | yes |
| <a name="input_enable_logging"></a> [enable\_logging](#input\_enable\_logging) | Enable access logging on the internal ALB backend service (L7 request logs -> Cloud Logging at full sampling; reaches the SIEM via the org all-load-balancer sink). Opt-in; default off so existing producers are unaffected. | `bool` | `false` | no |
| <a name="input_labels"></a> [labels](#input\_labels) | Labels to apply to resources that support them. | `map(string)` | `{}` | no |
| <a name="input_name"></a> [name](#input\_name) | Resource name prefix for the producer-side resources. | `string` | n/a | yes |
| <a name="input_network"></a> [network](#input\_network) | Self-link of the VPC network hosting the internal ALB frontend. | `string` | n/a | yes |
| <a name="input_project"></a> [project](#input\_project) | The project ID in which to create the producer-side PSC resources. | `string` | n/a | yes |
| <a name="input_proxy_only_subnet"></a> [proxy\_only\_subnet](#input\_proxy\_only\_subnet) | Self-link of the caller-created REGIONAL\_MANAGED\_PROXY subnet for this region. The module does not create this subnet; it is referenced only to order the ALB forwarding rule after the proxy-only subnet exists. | `string` | n/a | yes |
| <a name="input_psc_nat_subnets"></a> [psc\_nat\_subnets](#input\_psc\_nat\_subnets) | List of self-links of caller-created PRIVATE\_SERVICE\_CONNECT NAT subnets used by the service attachment. The module does not create these subnets. | `list(string)` | n/a | yes |
| <a name="input_region"></a> [region](#input\_region) | The region in which the Cloud Run service, internal ALB, and service attachment live. | `string` | n/a | yes |
| <a name="input_subnetwork"></a> [subnetwork](#input\_subnetwork) | Self-link of the subnetwork in which the internal ALB VIP is allocated. | `string` | n/a | yes |
| <a name="input_tls_certificates"></a> [tls\_certificates](#input\_tls\_certificates) | Certificate Manager regional certificate ids<br/>(projects/<project>/locations/<region>/certificates/<name>) for the<br/>frontend to serve. Empty (the default) keeps the plain-HTTP frontend on<br/>:80. Non-empty switches the frontend to a target HTTPS proxy on :443,<br/>which replaces the forwarding rule and so recreates the service<br/>attachment (closing every connected consumer endpoint for good: each<br/>consumer must then recreate its endpoint, see the README; the attachment<br/>name is unchanged). Certificates must already be ACTIVE:<br/>consumers dial the certificate's hostname and resolve it to their own<br/>PSC endpoint IP (a private DNS zone in the consumer VPC), so a regional<br/>Google-managed certificate with DNS authorization for a real domain<br/>verifies against public roots with no trust distribution. Attach it on<br/>the provision after the one that issued it. | `list(string)` | `[]` | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_internal_lb_ip"></a> [internal\_lb\_ip](#output\_internal\_lb\_ip) | Internal VIP of the regional internal ALB frontend. |
| <a name="output_service_attachment_id"></a> [service\_attachment\_id](#output\_service\_attachment\_id) | Self-link / id of the PSC service attachment. This is the value handed to the consumer module's service\_attachment input. |
<!-- END_TF_DOCS -->
