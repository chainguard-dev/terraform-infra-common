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
- Private Service Connect (PSC) support for cross-project access

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
| <a name="input_enable_private_path_for_google_cloud_services"></a> [enable\_private\_path\_for\_google\_cloud\_services](#input\_enable\_private\_path\_for\_google\_cloud\_services) | Enable access from Google Cloud services (e.g. BigQuery). | `bool` | `false` | no |
| <a name="input_insights_config"></a> [insights\_config](#input\_insights\_config) | Query Insights configuration for monitoring and troubleshooting database performance.<br/>    When set, enables Query Insights with the specified configuration:<br/>      • query\_string\_length: Maximum query length stored in bytes (256-4500 for ENTERPRISE, 1024-100000 for ENTERPRISE\_PLUS; default: 1024)<br/>      • query\_plans\_per\_minute: Number of query execution plans captured per minute (0-20 for ENTERPRISE, 0-200 for ENTERPRISE\_PLUS; default: 5)<br/>      • record\_application\_tags: Record application tags from queries (default: false)<br/>      • record\_client\_address: Record client IP addresses (default: false)<br/>    Set to null to disable Query Insights (default). | <pre>object({<br/>    query_string_length     = optional(number)<br/>    query_plans_per_minute  = optional(number)<br/>    record_application_tags = optional(bool)<br/>    record_client_address   = optional(bool)<br/>  })</pre> | `null` | no |
| <a name="input_labels"></a> [labels](#input\_labels) | The resource labels to represent user-provided metadata. | `map(string)` | `{}` | no |
| <a name="input_maintenance_window_day"></a> [maintenance\_window\_day](#input\_maintenance\_window\_day) | Day of week for maintenance window (1=Mon … 7=Sun, 0 for unspecified). | `number` | `7` | no |
| <a name="input_maintenance_window_hour"></a> [maintenance\_window\_hour](#input\_maintenance\_window\_hour) | Hour (0‑23 UTC) for the maintenance window. | `number` | `5` | no |
| <a name="input_name"></a> [name](#input\_name) | Cloud SQL instance name (lowercase letters, numbers, and hyphens; up to 98 characters). | `string` | n/a | yes |
| <a name="input_network"></a> [network](#input\_network) | Self‑link or name of the VPC network used for private IP connectivity. | `string` | n/a | yes |
| <a name="input_primary_zone"></a> [primary\_zone](#input\_primary\_zone) | Optional zone for the primary instance. | `string` | `null` | no |
| <a name="input_product"></a> [product](#input\_product) | Product label to apply to the service. | `string` | `"unknown"` | no |
| <a name="input_project"></a> [project](#input\_project) | GCP project ID hosting the Cloud SQL instance. | `string` | n/a | yes |
| <a name="input_psc_allowed_consumer_projects"></a> [psc\_allowed\_consumer\_projects](#input\_psc\_allowed\_consumer\_projects) | List of project IDs allowed to connect to this Cloud SQL instance via PSC. Only used when psc\_enabled is true. | `list(string)` | `[]` | no |
| <a name="input_psc_enabled"></a> [psc\_enabled](#input\_psc\_enabled) | Enable Private Service Connect (PSC) for cross-project access to the Cloud SQL instance. | `bool` | `false` | no |
| <a name="input_read_replica_regions"></a> [read\_replica\_regions](#input\_read\_replica\_regions) | List of regions in which to create read replicas. Empty list for none. | `list(string)` | `[]` | no |
| <a name="input_region"></a> [region](#input\_region) | GCP region for the primary instance. | `string` | n/a | yes |
| <a name="input_replicas_deletion_protection"></a> [replicas\_deletion\_protection](#input\_replicas\_deletion\_protection) | Enable deletion protection for read replicas. | `bool` | `false` | no |
| <a name="input_ssl_mode"></a> [ssl\_mode](#input\_ssl\_mode) | SSL mode for the Cloud SQL instance. Default is TRUSTED\_CLIENT\_CERTIFICATE\_REQUIRED. | `string` | `"TRUSTED_CLIENT_CERTIFICATE_REQUIRED"` | no |
| <a name="input_storage_gb"></a> [storage\_gb](#input\_storage\_gb) | Initial SSD storage size in GB. | `number` | `256` | no |
| <a name="input_team"></a> [team](#input\_team) | Team label to apply to resources (replaces deprecated 'squad'). | `string` | n/a | yes |
| <a name="input_tier"></a> [tier](#input\_tier) | Machine tier for the Cloud SQL instance. | `string` | `"db-perf-optimized-N-16"` | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_client_sa_bindings"></a> [client\_sa\_bindings](#output\_client\_sa\_bindings) | Map of service‑account email → IAM binding resource ID. |
| <a name="output_instance_connection_name"></a> [instance\_connection\_name](#output\_instance\_connection\_name) | Fully‑qualified connection name of the primary instance (<project>:<region>:<instance>). |
| <a name="output_instance_name"></a> [instance\_name](#output\_instance\_name) | Name of the primary Cloud SQL instance. |
| <a name="output_instance_self_link"></a> [instance\_self\_link](#output\_instance\_self\_link) | Self‑link URI of the primary Cloud SQL instance. |
| <a name="output_private_ip_address"></a> [private\_ip\_address](#output\_private\_ip\_address) | Private IPv4 address of the primary Cloud SQL instance. |
| <a name="output_psc_dns_name"></a> [psc\_dns\_name](#output\_psc\_dns\_name) | The DNS name to use for PSC connections. Only populated when PSC is enabled. |
| <a name="output_psc_service_attachment_link"></a> [psc\_service\_attachment\_link](#output\_psc\_service\_attachment\_link) | The PSC service attachment link for connecting from consumer projects. Only populated when PSC is enabled. |
| <a name="output_replica_connection_names"></a> [replica\_connection\_names](#output\_replica\_connection\_names) | Map of replica region → connection name. Empty if no replicas. |
| <a name="output_replica_private_ips"></a> [replica\_private\_ips](#output\_replica\_private\_ips) | Map of replica region → private IPv4 address. Empty if no replicas. |
<!-- END_TF_DOCS -->
