# `networking`

This module sets up GCP networking suitable for operating Cloud Run services
utilizing the preview
[Direct VPC egress](https://cloud.google.com/run/docs/configuring/vpc-direct-vpc)
feature to talk to other "internal ingress" Cloud Run services, and access other
GCP resources that live within or are accessible via the provisioned network.
The intended usage of this module:

```hcl
// Create a network with several regional subnets
module "networking" {
  source = "chainguard-dev/common/infra//modules/networking"

  name       = "my-networking"
  project_id = var.project_id

  // These are all of the regions where direct VPC egress is
  // supported in preview.
  regions = [
    "us-east1",
    "us-central1",
    "europe-west1",
    "europe-west3",
    "asia-northeast1",
  ]
}
```

<!-- BEGIN_TF_DOCS -->
## Requirements

No requirements.

## Providers

| Name | Version |
|------|---------|
| <a name="provider_google"></a> [google](#provider\_google) | n/a |
| <a name="provider_random"></a> [random](#provider\_random) | n/a |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [google_compute_network.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_network) | resource |
| [google_compute_route.egress-inet](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_route) | resource |
| [google_compute_subnetwork.regional](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_subnetwork) | resource |
| [google_dns_managed_zone.cloud-run-internal](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/dns_managed_zone) | resource |
| [google_dns_managed_zone.private-google-apis](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/dns_managed_zone) | resource |
| [google_dns_policy.dns_logging_policy](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/dns_policy) | resource |
| [google_dns_record_set.cloud-run-cname](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/dns_record_set) | resource |
| [google_dns_record_set.private-googleapis-a-record](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/dns_record_set) | resource |
| [random_string.suffix](https://registry.terraform.io/providers/hashicorp/random/latest/docs/resources/string) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_cidr"></a> [cidr](#input\_cidr) | n/a | `string` | `"10.0.0.0/8"` | no |
| <a name="input_hosted_zone_logging_enabled"></a> [hosted\_zone\_logging\_enabled](#input\_hosted\_zone\_logging\_enabled) | Whether or not to enable DNS Hosted Zone Cloud Logging | `bool` | `true` | no |
| <a name="input_labels"></a> [labels](#input\_labels) | Labels to apply to the networking resources. | `map(string)` | `{}` | no |
| <a name="input_name"></a> [name](#input\_name) | n/a | `string` | n/a | yes |
| <a name="input_netnum_offset"></a> [netnum\_offset](#input\_netnum\_offset) | cidrsubnet netnum offset for the subnet. See https://developer.hashicorp.com/terraform/language/functions/cidrsubnet for more details | `number` | `0` | no |
| <a name="input_product"></a> [product](#input\_product) | Product label to apply to the service. | `string` | `"unknown"` | no |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | n/a | `string` | n/a | yes |
| <a name="input_regions"></a> [regions](#input\_regions) | The list of regions in which to provision subnets suitable for use with Cloud Run direct VPC egress. | `list(string)` | n/a | yes |
| <a name="input_require_squad"></a> [require\_squad](#input\_require\_squad) | Whether to require squad variable to be specified | `bool` | `false` | no |
| <a name="input_squad"></a> [squad](#input\_squad) | Squad label to apply to the networking resources. | `string` | `""` | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_network_id"></a> [network\_id](#output\_network\_id) | n/a |
| <a name="output_regional-networks"></a> [regional-networks](#output\_regional-networks) | n/a |
<!-- END_TF_DOCS -->
