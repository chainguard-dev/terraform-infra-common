# `bastion`

A hardened IAP-only jump-host. It comes with Google Cloud Ops Agent,
`auditd`, weekly OS patching, and optionally the **Cloud SQL Auth Proxy v2
binary** for database (MySQL/PostgreSQL) connectivity when needed.

## Key Features

* **Zero-trust SSH** – IAP tunnel + OS Login (no public IP)
* **Hardened VM** – Shielded VM, secure-boot, auditd, Ops Agent metrics & logs
* **Optional Cloud SQL support** – Installs Cloud SQL Auth Proxy v2 binary
  when `install_sql_proxy = true`
* **Automatic patching** – Weekly OS Config patch deployment with reboot
* **Dedicated NAT** – Optional Cloud NAT router for outbound security updates

## Quick Start

### Basic jump-host (no Cloud SQL)
```hcl
module "bastion" {
  source = "github.com/chainguard-dev/terraform-infra-common//modules/bastion"

  name       = "jump-host"
  project_id = "my-project"
  zone       = "us-central1-a"
  network    = "projects/my-project/global/networks/my-vpc"
  subnetwork = "my-private-subnet"
  squad      = "platform"

  # IAM principals granted OS Login access
  dev_principals = ["group:engineering@my-company.com"]

  # No Cloud SQL tooling (default)
  install_sql_proxy = false
}
```

### Jump-host with Cloud SQL Auth Proxy
```hcl
module "bastion" {
  source = "github.com/chainguard-dev/terraform-infra-common//modules/bastion"

  name       = "sql-bastion"
  project_id = "my-project"
  zone       = "us-central1-a"
  network    = "projects/my-project/global/networks/my-vpc"
  subnetwork = "my-private-subnet"
  squad      = "platform"

  # IAM principals granted OS Login & Cloud SQL access
  dev_principals = ["group:db-breakglass@my-company.com"]

  # Install the Cloud SQL Auth Proxy binary
  install_sql_proxy = true
}
```

## Security Model

### Access Control
- **IAP tunnel**: All SSH connections go through Identity-Aware Proxy
- **OS Login**: Users must have `compute.osLogin` role
- **Cloud SQL IAM** (optional): When enabled, users need `cloudsql.client` and
  `cloudsql.instanceUser` roles
- **Minimal SA** The bastion runs a minimal SA for metrics and logging purposes.
  This SA is not granted `cloudsql` roles. Human users connect via Group-based IAM.

### Auditing & Monitoring
- **Command auditing**: `auditd` logs all executed commands
- **Cloud Logging**: Startup scripts and system events
- **Cloud Monitoring**: VM metrics via Ops Agent
- **Secure boot**: Shielded VM prevents unauthorized boot modifications

### Network Security
- **No public IP**: VM only has private IP address
- **IAP-only SSH**: Firewall only allows SSH from IAP IP ranges (`35.235.240.0/20`)
- **Optional NAT**: Dedicated Cloud NAT for outbound package updates

### Automatic Patching
- **Schedule**: Weekly patching on configurable day/time (default: Monday 03:00 UTC)
- **Reboot**: Always reboots after patching to ensure kernel updates
- **OS Config**: Uses Google Cloud OS Config for patch management

## Using the Auth Proxy

SSH into the bastion and auth with gcloud:

```bash
$ gcloud compute ssh sql-bastion --tunnel-through-iap --project=<PROJECT> --zone=<ZONE>

user@bastion $ gcloud auth application-default login --quiet
```

Start the Auth Proxy on the bastion:

```bash
user@bastion $ cloud-sql-proxy --auto-iam-authn --private-ip <PROJECT>:<REGION>:<INSTANCE ID>
```

In a different terminal tab, open a tunnel:

```bash
$ gcloud compute ssh sql-bastion --tunnel-through-iap --project=<PROJECT> --zone=<ZONE> -- -N -L 15432:127.0.0.1:5432
```

In a third terminal tab, you can now connect to the DB:

```bash
$ psql "host=127.0.0.1 port=15432 user=<YOU>@chainguard.dev dbname=<DB NAME>"  # Postgres example
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
| [google_iap_tunnel_instance_iam_member.dev_iap_tunnel_resource_accessor](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/iap_tunnel_instance_iam_member) | resource |
| [google_os_config_patch_deployment.bastion_patching](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/os_config_patch_deployment) | resource |
| [google_project_iam_member.bastion_extra_roles](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/project_iam_member) | resource |
| [google_project_iam_member.bastion_sa_log_writer](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/project_iam_member) | resource |
| [google_project_iam_member.bastion_sa_metric_writer](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/project_iam_member) | resource |
| [google_project_iam_member.dev_cloudsql_client](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/project_iam_member) | resource |
| [google_project_iam_member.dev_cloudsql_instance_user](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/project_iam_member) | resource |
| [google_project_iam_member.dev_os_login](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/project_iam_member) | resource |
| [google_project_iam_member.dev_service_account_user](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/project_iam_member) | resource |
| [google_project_service.os_config_api](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/project_service) | resource |
| [google_service_account.bastion_sa](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/service_account) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_deletion_protection"></a> [deletion\_protection](#input\_deletion\_protection) | GCE API deletion protection flag. When true, prevents instance deletion via the API. | `bool` | `true` | no |
| <a name="input_dev_principals"></a> [dev\_principals](#input\_dev\_principals) | IAM principals (users, groups, or service accounts) granted OS Login & Cloud SQL access. | `list(string)` | n/a | yes |
| <a name="input_enable_nat"></a> [enable\_nat](#input\_enable\_nat) | Whether to create a dedicated Cloud NAT router for outbound egress. Disable when VPC already has NAT. | `bool` | `true` | no |
| <a name="input_extra_sa_roles"></a> [extra\_sa\_roles](#input\_extra\_sa\_roles) | Additional IAM roles to bind to the bastion's service account. | `list(string)` | `[]` | no |
| <a name="input_install_sql_proxy"></a> [install\_sql\_proxy](#input\_install\_sql\_proxy) | Whether to install the Cloud SQL Auth Proxy binary and grant associated IAM permissions. | `bool` | `false` | no |
| <a name="input_machine_type"></a> [machine\_type](#input\_machine\_type) | Compute Engine machine type for the bastion. | `string` | `"e2-micro"` | no |
| <a name="input_name"></a> [name](#input\_name) | Name prefix for all resources (also used as network tag). | `string` | n/a | yes |
| <a name="input_network"></a> [network](#input\_network) | VPC network self-link or name the bastion joins. | `string` | n/a | yes |
| <a name="input_patch_day"></a> [patch\_day](#input\_patch\_day) | Day of week (in UTC) when OS Config patching runs. | `string` | `"MONDAY"` | no |
| <a name="input_patch_time_utc"></a> [patch\_time\_utc](#input\_patch\_time\_utc) | Time of day in HH:MM (UTC) when patching runs. | `string` | `"03:00"` | no |
| <a name="input_product"></a> [product](#input\_product) | Product label to apply to the service. | `string` | `"unknown"` | no |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | Project in which to deploy the bastion host. | `string` | n/a | yes |
| <a name="input_squad"></a> [squad](#input\_squad) | Squad or team label applied to the instance (required). | `string` | n/a | yes |
| <a name="input_startup_script"></a> [startup\_script](#input\_startup\_script) | additional startup script snippet to execute on bastion. | `string` | `""` | no |
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
