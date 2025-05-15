# Google Cloud SQL PostgreSQL Module

Opinionated Terraform module that provisions a Cloud SQL for PostgreSQL instance.

## Features

- Creates a Cloud SQL PostgreSQL instance (defaults to PostgreSQL 17)
- Private IP connectivity only (no public IP exposure)
- Configurable high availability with regional failover
- Cross-region read replica support
- Automated backups and point-in-time recovery
- Configurable maintenance windows
- IAM authentication enabled by default
- Service account access management
- Storage autoresizing with configurable limits

## Usage example

```hcl
module "postgres" {
  source = "github.com/chainguard-dev/terraform-infra-common//modules/cloudsql-postgres"
  name = "myapp-db"
  project = "my-gcp-project"
  region = "us-central1"
  network = "projects/my-gcp-project/global/networks/my-vpc"
  squad = "platform"

  # Optional configurations
  enable_high_availability = true
  read_replica_regions = ["us-east1"]
  tier = "db-perf-optimized-N-16"

  authorized_client_service_accounts = [
    "myapp@my-gcp-project.iam.gserviceaccount.com"
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

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [google_project_iam_member.client_sa](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/project_iam_member) | resource |
| [google_sql_database_instance.replicas](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/sql_database_instance) | resource |
| [google_sql_database_instance.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/sql_database_instance) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_authorized_client_service_accounts"></a> [authorized\_client\_service\_accounts](#input\_authorized\_client\_service\_accounts) | List of Google service account emails that require `roles/cloudsql.client`. | `list(string)` | `[]` | no |
| <a name="input_backup_enabled"></a> [backup\_enabled](#input\_backup\_enabled) | Enable automated daily backups. | `bool` | `true` | no |
| <a name="input_backup_start_time"></a> [backup\_start\_time](#input\_backup\_start\_time) | Start time for the backup window (UTC, HH:MM). | `string` | `"08:00"` | no |
| <a name="input_database_flags"></a> [database\_flags](#input\_database\_flags) | Additional database flags to set on the instance. | `map(string)` | `{}` | no |
| <a name="input_database_version"></a> [database\_version](#input\_database\_version) | Cloud SQL engine version. | `string` | `"POSTGRES_17"` | no |
| <a name="input_deletion_protection"></a> [deletion\_protection](#input\_deletion\_protection) | Cloud SQL API deletion protection flag. When true, prevents instance deletion via the API. | `bool` | `true` | no |
| <a name="input_disk_autoresize_limit"></a> [disk\_autoresize\_limit](#input\_disk\_autoresize\_limit) | Upper GB limit for automatic storage growth. Set to 0 for unlimited. | `number` | `4096` | no |
| <a name="input_edition"></a> [edition](#input\_edition) | Cloud SQL edition for the instance. Accepted values:<br/>      • "ENTERPRISE"<br/>      • "ENTERPRISE\_PLUS" | `string` | `null` | no |
| <a name="input_enable_high_availability"></a> [enable\_high\_availability](#input\_enable\_high\_availability) | Enable regional high‑availability (REGIONAL availability\_type). | `bool` | `false` | no |
| <a name="input_enable_point_in_time_recovery"></a> [enable\_point\_in\_time\_recovery](#input\_enable\_point\_in\_time\_recovery) | Enable point-in-time recovery (continuous WAL archiving). | `bool` | `true` | no |
| <a name="input_maintenance_window_day"></a> [maintenance\_window\_day](#input\_maintenance\_window\_day) | Day of week for maintenance window (1=Mon … 7=Sun, 0 for unspecified). | `number` | `7` | no |
| <a name="input_maintenance_window_hour"></a> [maintenance\_window\_hour](#input\_maintenance\_window\_hour) | Hour (0‑23 UTC) for the maintenance window. | `number` | `5` | no |
| <a name="input_name"></a> [name](#input\_name) | Cloud SQL instance name (lowercase letters, numbers, and hyphens; up to 98 characters). | `string` | n/a | yes |
| <a name="input_network"></a> [network](#input\_network) | Self‑link or name of the VPC network used for private IP connectivity. | `string` | n/a | yes |
| <a name="input_primary_zone"></a> [primary\_zone](#input\_primary\_zone) | Optional zone for the primary instance. | `string` | `null` | no |
| <a name="input_project"></a> [project](#input\_project) | GCP project ID hosting the Cloud SQL instance. | `string` | n/a | yes |
| <a name="input_read_replica_regions"></a> [read\_replica\_regions](#input\_read\_replica\_regions) | List of regions in which to create read replicas. Empty list for none. | `list(string)` | `[]` | no |
| <a name="input_region"></a> [region](#input\_region) | GCP region for the primary instance. | `string` | n/a | yes |
| <a name="input_squad"></a> [squad](#input\_squad) | Squad or team label applied to the instance (required). | `string` | n/a | yes |
| <a name="input_storage_gb"></a> [storage\_gb](#input\_storage\_gb) | Initial SSD storage size in GB. | `number` | `256` | no |
| <a name="input_tier"></a> [tier](#input\_tier) | Machine tier for the Cloud SQL instance. | `string` | `"db-perf-optimized-N-16"` | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_client_sa_bindings"></a> [client\_sa\_bindings](#output\_client\_sa\_bindings) | Map of service‑account email → IAM binding resource ID. |
| <a name="output_instance_connection_name"></a> [instance\_connection\_name](#output\_instance\_connection\_name) | Fully‑qualified connection name of the primary instance (<project>:<region>:<instance>). |
| <a name="output_instance_name"></a> [instance\_name](#output\_instance\_name) | Name of the primary Cloud SQL instance. |
| <a name="output_instance_self_link"></a> [instance\_self\_link](#output\_instance\_self\_link) | Self‑link URI of the primary Cloud SQL instance. |
| <a name="output_private_ip_address"></a> [private\_ip\_address](#output\_private\_ip\_address) | Private IPv4 address of the primary Cloud SQL instance. |
| <a name="output_replica_connection_names"></a> [replica\_connection\_names](#output\_replica\_connection\_names) | Map of replica region → connection name. Empty if no replicas. |
| <a name="output_replica_private_ips"></a> [replica\_private\_ips](#output\_replica\_private\_ips) | Map of replica region → private IPv4 address. Empty if no replicas. |
<!-- END_TF_DOCS -->
