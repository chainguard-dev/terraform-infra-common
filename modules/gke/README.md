<!-- BEGIN_TF_DOCS -->
## Requirements

No requirements.

## Providers

| Name | Version |
|------|---------|
| <a name="provider_google"></a> [google](#provider\_google) | n/a |
| <a name="provider_google-beta"></a> [google-beta](#provider\_google-beta) | n/a |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [google-beta_google_container_node_pool.pools](https://registry.terraform.io/providers/hashicorp/google-beta/latest/docs/resources/google_container_node_pool) | resource |
| [google_compute_firewall.master_webhook](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_firewall) | resource |
| [google_container_cluster.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/container_cluster) | resource |
| [google_project_iam_member.cluster](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/project_iam_member) | resource |
| [google_service_account.cluster_default](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/service_account) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_cluster_autoscaling"></a> [cluster\_autoscaling](#input\_cluster\_autoscaling) | Enabling of node auto-provisioning | `bool` | `false` | no |
| <a name="input_cluster_autoscaling_cpu_limits"></a> [cluster\_autoscaling\_cpu\_limits](#input\_cluster\_autoscaling\_cpu\_limits) | cluster autoscaling cpu limits | <pre>object({<br/>    resource_type = optional(string, "cpu")<br/>    minimum       = optional(number, 4)<br/>    maximum       = optional(number, 10)<br/>  })</pre> | `{}` | no |
| <a name="input_cluster_autoscaling_memory_limits"></a> [cluster\_autoscaling\_memory\_limits](#input\_cluster\_autoscaling\_memory\_limits) | cluster autoscaling memory limits | <pre>object({<br/>    resource_type = optional(string, "memory"),<br/>    minimum       = optional(number, 8)<br/>    maximum       = optional(number, 80)<br/>  })</pre> | `null` | no |
| <a name="input_cluster_autoscaling_profile"></a> [cluster\_autoscaling\_profile](#input\_cluster\_autoscaling\_profile) | cluster autoscaling profile | `string` | `null` | no |
| <a name="input_cluster_autoscaling_provisioning_defaults"></a> [cluster\_autoscaling\_provisioning\_defaults](#input\_cluster\_autoscaling\_provisioning\_defaults) | cluster autoscaling provisioning defaults | <pre>object({<br/>    disk_size = optional(number, null)<br/>    disk_type = optional(string, null)<br/>    shielded_instance_config = optional(object({<br/>      enable_secure_boot          = optional(bool, null)<br/>      enable_integrity_monitoring = optional(bool, null)<br/>    }), null)<br/>    management = optional(object({<br/>      auto_upgrade = optional(bool, null)<br/>      auto_repair  = optional(bool, null)<br/>    }), null)<br/>  })</pre> | `null` | no |
| <a name="input_deletion_protection"></a> [deletion\_protection](#input\_deletion\_protection) | Toggle to prevent accidental deletion of resources. | `bool` | `true` | no |
| <a name="input_enable_private_nodes"></a> [enable\_private\_nodes](#input\_enable\_private\_nodes) | Enable private nodes by default | `bool` | `false` | no |
| <a name="input_extra_roles"></a> [extra\_roles](#input\_extra\_roles) | Extra roles to add to the cluster's default service account | `map(string)` | `{}` | no |
| <a name="input_labels"></a> [labels](#input\_labels) | Labels to apply to the gke resources. | `map(string)` | `{}` | no |
| <a name="input_master_ipv4_cidr_block"></a> [master\_ipv4\_cidr\_block](#input\_master\_ipv4\_cidr\_block) | If specified, will use this CIDR block for the master's IP address | `string` | n/a | yes |
| <a name="input_name"></a> [name](#input\_name) | n/a | `string` | n/a | yes |
| <a name="input_network"></a> [network](#input\_network) | The network to deploy the cluster in. | `string` | n/a | yes |
| <a name="input_pools"></a> [pools](#input\_pools) | n/a | <pre>map(object({<br/>    min_node_count                    = optional(number, 1)<br/>    max_node_count                    = optional(number, 1)<br/>    machine_type                      = optional(string, "c3-standard-4")<br/>    disk_type                         = optional(string, "pd-balanced")<br/>    disk_size                         = optional(number, 100)<br/>    ephemeral_storage_local_ssd_count = optional(number, 0)<br/>    spot                              = optional(bool, false)<br/>    gvisor                            = optional(bool, false)<br/>    labels                            = optional(map(string), {})<br/>    taints = optional(list(object({<br/>      key    = string<br/>      value  = string<br/>      effect = string<br/>    })), [])<br/>    network_config = optional(object({<br/>      enable_private_nodes = optional(bool, false)<br/>      create_pod_range     = optional(bool, true)<br/>      pod_ipv4_cidr_block  = optional(string, null)<br/>    }), null)<br/>  }))</pre> | n/a | yes |
| <a name="input_project"></a> [project](#input\_project) | n/a | `string` | n/a | yes |
| <a name="input_region"></a> [region](#input\_region) | Always create a regional cluster since GKE doesn't charge differently for regional/zonal clusters. Rather, we configure the node locations using `var.zones` | `any` | n/a | yes |
| <a name="input_release_channel"></a> [release\_channel](#input\_release\_channel) | GKE release channel | `string` | `"REGULAR"` | no |
| <a name="input_require_squad"></a> [require\_squad](#input\_require\_squad) | Whether to require squad variable to be specified | `bool` | `true` | no |
| <a name="input_resource_usage_export_config"></a> [resource\_usage\_export\_config](#input\_resource\_usage\_export\_config) | Config for exporting resource usage. | <pre>object({<br/>    bigquery_dataset_id                  = optional(string, "")<br/>    enable_network_egress_metering       = optional(bool, false)<br/>    enable_resource_consumption_metering = optional(bool, true)<br/>  })</pre> | n/a | yes |
| <a name="input_squad"></a> [squad](#input\_squad) | squad label to apply to the service. | `string` | `""` | no |
| <a name="input_subnetwork"></a> [subnetwork](#input\_subnetwork) | The subnetwork to deploy the cluster in. | `string` | n/a | yes |
| <a name="input_zones"></a> [zones](#input\_zones) | If specified, will spread nodes across these zones | `list(string)` | `null` | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_cluster_ca_certificate"></a> [cluster\_ca\_certificate](#output\_cluster\_ca\_certificate) | n/a |
| <a name="output_cluster_endpoint"></a> [cluster\_endpoint](#output\_cluster\_endpoint) | n/a |
| <a name="output_cluster_id"></a> [cluster\_id](#output\_cluster\_id) | n/a |
| <a name="output_cluster_name"></a> [cluster\_name](#output\_cluster\_name) | n/a |
| <a name="output_service_account_email"></a> [service\_account\_email](#output\_service\_account\_email) | n/a |
<!-- END_TF_DOCS -->
