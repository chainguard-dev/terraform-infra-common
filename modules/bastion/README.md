# `bastion`

A hardened, secure-by-default bastion host.

Key features:

* OS-Login + IAP-only SSH (no exposed public IP)
* Shielded-VM secure-boot, vTPM & integrity monitoring
* Google Cloud Ops Agent for logs / metrics
* `auditd` captures every executed command
* Weekly OS Config patching (configurable day/time)
* Least-privilege service-account & scopes
* Optional Cloud SQL (PostgreSQL) local proxy
* Optional dedicated Cloud NAT for outbound package updates

## Example Usage
```hcl
module "bastion" {
  source = "github.com/chainguard-dev/terraform-infra-common//modules/modules/bastion"

  name               = "sql-bastion"
  project_id         = var.project_id
  zone               = "us-central1-a"
  network            = module.networking.network_id
  subnetwork         = module.networking.private_subnet
  dev_users          = ["alice@example.com", "bob@example.com"]
  postgres_conn_name = "myproj:us-central1:db01"          # optional
}
```

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
| [google_compute_firewall.iap_ssh](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_firewall) | resource |
| [google_compute_instance.bastion](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_instance) | resource |
| [google_compute_router.nat_router](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_router) | resource |
| [google_compute_router_nat.bastion_nat](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_router_nat) | resource |
| [google_os_config_patch_deployment.bastion_patching](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/os_config_patch_deployment) | resource |
| [google_project_iam_member.bastion_extra_roles](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/project_iam_member) | resource |
| [google_project_iam_member.bastion_sa_cloudsql_client](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/project_iam_member) | resource |
| [google_project_iam_member.bastion_sa_cloudsql_instance_user](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/project_iam_member) | resource |
| [google_project_iam_member.bastion_sa_log_writer](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/project_iam_member) | resource |
| [google_project_iam_member.bastion_sa_metric_writer](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/project_iam_member) | resource |
| [google_project_iam_member.dev_cloudsql_client](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/project_iam_member) | resource |
| [google_project_iam_member.dev_cloudsql_instance_user](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/project_iam_member) | resource |
| [google_project_iam_member.dev_os_login](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/project_iam_member) | resource |
| [google_project_service.os_config_api](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/project_service) | resource |
| [google_service_account.bastion_sa](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/service_account) | resource |
| [google_sql_user.bastion_sa_db_user](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/sql_user) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_deletion_protection"></a> [deletion\_protection](#input\_deletion\_protection) | GCE API deletion protection flag. When true, prevents instance deletion via the API. | `bool` | `true` | no |
| <a name="input_dev_users"></a> [dev\_users](#input\_dev\_users) | List of developer email addresses granted OS Login & Cloud SQL access. | `list(string)` | n/a | yes |
| <a name="input_enable_nat"></a> [enable\_nat](#input\_enable\_nat) | Whether to create a dedicated Cloud NAT router for outbound egress. Disable when VPC already has NAT. | `bool` | `true` | no |
| <a name="input_extra_sa_roles"></a> [extra\_sa\_roles](#input\_extra\_sa\_roles) | Additional IAM roles to bind to the bastion's service account. | `list(string)` | `[]` | no |
| <a name="input_machine_type"></a> [machine\_type](#input\_machine\_type) | Compute Engine machine type for the bastion. | `string` | `"e2-micro"` | no |
| <a name="input_name"></a> [name](#input\_name) | Name prefix for all resources (also used as network tag). | `string` | n/a | yes |
| <a name="input_network"></a> [network](#input\_network) | VPC network self-link or name the bastion joins. | `string` | n/a | yes |
| <a name="input_patch_day"></a> [patch\_day](#input\_patch\_day) | Day of week (in UTC) when OS Config patching runs. | `string` | `"MONDAY"` | no |
| <a name="input_patch_time_utc"></a> [patch\_time\_utc](#input\_patch\_time\_utc) | Time of day in HH:MM (UTC) when patching runs. | `string` | `"03:00"` | no |
| <a name="input_postgres_conn_name"></a> [postgres\_conn\_name](#input\_postgres\_conn\_name) | Cloud SQL connection name <project>:<region>:<instance>. When empty, the Postgres proxy is not installed. | `string` | `""` | no |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | Project in which to deploy the bastion host. | `string` | n/a | yes |
| <a name="input_squad"></a> [squad](#input\_squad) | Squad or team label applied to the instance (required). | `string` | n/a | yes |
| <a name="input_subnetwork"></a> [subnetwork](#input\_subnetwork) | Subnetwork name the bastion joins (must be private). | `string` | n/a | yes |
| <a name="input_zone"></a> [zone](#input\_zone) | Compute Engine zone for the bastion VM (e.g. us-central1-a). | `string` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_instance_name"></a> [instance\_name](#output\_instance\_name) | Name of the bastion compute instance |
| <a name="output_internal_ip"></a> [internal\_ip](#output\_internal\_ip) | Internal IP address of the bastion VM |
| <a name="output_nat_router_name"></a> [nat\_router\_name](#output\_nat\_router\_name) | Name of the Cloud NAT router (empty when enable\_nat = false) |
| <a name="output_service_account_email"></a> [service\_account\_email](#output\_service\_account\_email) | Service account email used by the bastion |
| <a name="output_ssh_target_tag"></a> [ssh\_target\_tag](#output\_ssh\_target\_tag) | Network tag applied to the bastion for SSH firewall rules |
<!-- END_TF_DOCS -->
