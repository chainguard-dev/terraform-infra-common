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
| [google_compute_firewall.master_webhook](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_firewall) | resource |
| [google_container_cluster.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/container_cluster) | resource |
| [google_project_iam_member.cluster](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/project_iam_member) | resource |
| [google_service_account.cluster_default](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/service_account) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_deletion_protection"></a> [deletion\_protection](#input\_deletion\_protection) | Toggle to prevent accidental deletion of resources. | `bool` | `true` | no |
| <a name="input_enable_private_nodes"></a> [enable\_private\_nodes](#input\_enable\_private\_nodes) | Enable private nodes by default | `bool` | `false` | no |
| <a name="input_extra_roles"></a> [extra\_roles](#input\_extra\_roles) | Extra roles to add to the cluster's default service account | `map(string)` | `{}` | no |
| <a name="input_master_ipv4_cidr_block"></a> [master\_ipv4\_cidr\_block](#input\_master\_ipv4\_cidr\_block) | If specified, will use this CIDR block for the master's IP address | `any` | n/a | yes |
| <a name="input_name"></a> [name](#input\_name) | n/a | `any` | n/a | yes |
| <a name="input_network"></a> [network](#input\_network) | n/a | `any` | n/a | yes |
| <a name="input_project"></a> [project](#input\_project) | n/a | `any` | n/a | yes |
| <a name="input_region"></a> [region](#input\_region) | Always create a regional cluster since GKE doesn't charge differently for regional/zonal clusters. Rather, we configure the node locations using `var.zones` | `any` | n/a | yes |
| <a name="input_release_channel"></a> [release\_channel](#input\_release\_channel) | GKE release channel | `string` | `"REGULAR"` | no |
| <a name="input_require_squad"></a> [require\_squad](#input\_require\_squad) | Whether to require squad variable to be specified | `bool` | `true` | no |
| <a name="input_squad"></a> [squad](#input\_squad) | squad label to apply to the service. | `string` | `""` | no |
| <a name="input_subnetwork"></a> [subnetwork](#input\_subnetwork) | n/a | `any` | n/a | yes |
| <a name="input_zones"></a> [zones](#input\_zones) | If specified, will spread nodes across these zones | `any` | `null` | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_cluster_ca_certificate"></a> [cluster\_ca\_certificate](#output\_cluster\_ca\_certificate) | n/a |
| <a name="output_cluster_endpoint"></a> [cluster\_endpoint](#output\_cluster\_endpoint) | n/a |
| <a name="output_cluster_id"></a> [cluster\_id](#output\_cluster\_id) | n/a |
| <a name="output_cluster_name"></a> [cluster\_name](#output\_cluster\_name) | n/a |
| <a name="output_service_account_email"></a> [service\_account\_email](#output\_service\_account\_email) | n/a |
<!-- END_TF_DOCS -->
