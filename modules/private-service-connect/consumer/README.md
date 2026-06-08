# `private-service-connect/consumer`

This submodule creates a **Private Service Connect (PSC) endpoint** — a
forwarding rule targeting a producer's service attachment — with an internal IP
in the consumer VPC.

The resource graph is:

```
internal IP (reserved, optional)
  -> PSC endpoint forwarding rule (load_balancing_scheme = "")
     targeting the producer's service attachment
```

If `address` is left empty the module reserves an `INTERNAL` IP in the supplied
subnetwork; otherwise the caller's pre-reserved address is used. DNS records for
the endpoint are intentionally out of scope (they are environment-specific) — the
module only exposes `endpoint_ip` so the caller can wire up DNS itself.

Supply the producer's `service_attachment_id` output as the `service_attachment`
input. See the parent module's [README](../README.md) for the producer ->
consumer flow and the two-phase apply.

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
| [google_compute_address.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_address) | resource |
| [google_compute_forwarding_rule.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_forwarding_rule) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_address"></a> [address](#input\_address) | Optional pre-reserved internal IP address (self-link / id) for the PSC endpoint. If empty, the module reserves an internal IP from the subnetwork. | `string` | `""` | no |
| <a name="input_allow_psc_global_access"></a> [allow\_psc\_global\_access](#input\_allow\_psc\_global\_access) | Allow clients in any region to reach this PSC endpoint. Leave false when every caller runs in the endpoint's region; set true when callers run in other regions (e.g. a multi-region Cloud Run service dialing this single-region endpoint), otherwise their connections are silently dropped at the PSC layer. | `bool` | `false` | no |
| <a name="input_name"></a> [name](#input\_name) | Resource name prefix for the consumer-side resources. | `string` | n/a | yes |
| <a name="input_network"></a> [network](#input\_network) | Self-link of the consumer VPC network hosting the PSC endpoint. | `string` | n/a | yes |
| <a name="input_project"></a> [project](#input\_project) | The project ID in which to create the consumer-side PSC endpoint. | `string` | n/a | yes |
| <a name="input_region"></a> [region](#input\_region) | The region in which to create the PSC endpoint. Must match the producer's service attachment region. | `string` | n/a | yes |
| <a name="input_service_attachment"></a> [service\_attachment](#input\_service\_attachment) | Self-link of the producer's PSC service attachment to target (the producer module's service\_attachment\_id output). | `string` | n/a | yes |
| <a name="input_subnetwork"></a> [subnetwork](#input\_subnetwork) | Self-link / id of the consumer subnetwork in which the endpoint's internal IP is allocated. | `string` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_endpoint_ip"></a> [endpoint\_ip](#output\_endpoint\_ip) | Internal IP address assigned to the PSC endpoint. |
| <a name="output_psc_connection_id"></a> [psc\_connection\_id](#output\_psc\_connection\_id) | The PSC connection id of the endpoint forwarding rule. |
<!-- END_TF_DOCS -->
