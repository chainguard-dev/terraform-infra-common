# BigQuery Log Sink Module

Terraform module for creating BigQuery datasets with Cloud Logging sinks for structured log ingestion.

## Features

- Creates BigQuery dataset and tables with configurable schemas
- Sets up Cloud Logging sinks to route logs to BigQuery
- Supports multiple tables with independent log filters
- Optional monitoring alerts for log ingestion health

## Usage

```hcl
module "bigquery_log_sink" {
  source = "../../terraform/public-modules/modules/bigquery-logsink"

  project_id                 = var.project_id
  name                       = "my_service"
  location                   = "US"
  partition_expiration_days  = 30  # Global retention for all tables

  team    = "platform"
  product = "logging"

  tables = {
    logs = {
      schema = jsonencode([
        { name = "timestamp", type = "TIMESTAMP", mode = "REQUIRED" },
        { name = "severity", type = "STRING", mode = "NULLABLE" },
        { name = "message", type = "STRING", mode = "NULLABLE" }
      ])
      partition_field   = "timestamp"
      clustering_fields = ["severity"]
      log_filter        = "resource.type=\"cloud_run_revision\" AND severity>=INFO"
      description       = "Application logs"
    }
  }

  deletion_protection = true
  enable_monitoring   = false
}
```

## Variables

See `variables.tf` for all available configuration options.

## Outputs

- `dataset_id` - BigQuery dataset ID
- `table_ids` - Map of table names to IDs
- `sink_names` - Map of log sink names
- `sink_writer_identities` - Map of sink writer service accounts

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
| [google_bigquery_dataset.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/bigquery_dataset) | resource |
| [google_bigquery_dataset_iam_member.sink_writers](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/bigquery_dataset_iam_member) | resource |
| [google_bigquery_table.tables](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/bigquery_table) | resource |
| [google_logging_project_sink.sinks](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/logging_project_sink) | resource |
| [google_monitoring_alert_policy.log_ingestion](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/monitoring_alert_policy) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_alert_auto_close_days"></a> [alert\_auto\_close\_days](#input\_alert\_auto\_close\_days) | Days after which to auto-close resolved alerts | `number` | `1` | no |
| <a name="input_alert_threshold_minutes"></a> [alert\_threshold\_minutes](#input\_alert\_threshold\_minutes) | Minutes without log ingestion before triggering alert | `number` | `180` | no |
| <a name="input_dataset_description"></a> [dataset\_description](#input\_dataset\_description) | Description of the BigQuery dataset | `string` | `""` | no |
| <a name="input_delete_contents_on_destroy"></a> [delete\_contents\_on\_destroy](#input\_delete\_contents\_on\_destroy) | Whether to delete dataset contents when destroying the dataset | `bool` | `false` | no |
| <a name="input_deletion_protection"></a> [deletion\_protection](#input\_deletion\_protection) | Enable deletion protection on tables | `bool` | `true` | no |
| <a name="input_enable_monitoring"></a> [enable\_monitoring](#input\_enable\_monitoring) | Enable monitoring alert policies for log ingestion | `bool` | `false` | no |
| <a name="input_labels"></a> [labels](#input\_labels) | Additional labels to apply to resources | `map(string)` | `{}` | no |
| <a name="input_location"></a> [location](#input\_location) | BigQuery dataset location | `string` | `"US"` | no |
| <a name="input_name"></a> [name](#input\_name) | Base name for the BigQuery resources | `string` | n/a | yes |
| <a name="input_notification_channels"></a> [notification\_channels](#input\_notification\_channels) | List of notification channel IDs for alerts | `list(string)` | `[]` | no |
| <a name="input_partition_expiration_days"></a> [partition\_expiration\_days](#input\_partition\_expiration\_days) | Global retention period in days for all table partitions | `number` | `30` | no |
| <a name="input_product"></a> [product](#input\_product) | Product label for resources | `string` | `null` | no |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | The GCP project ID where resources will be created | `string` | n/a | yes |
| <a name="input_tables"></a> [tables](#input\_tables) | Map of tables to create. Each key is the table name, and the value is an object with:<br/>- schema: JSON-encoded BigQuery schema<br/>- partition\_field: Field name for time partitioning (required)<br/>- clustering\_fields: List of fields for clustering (optional)<br/>- log\_filter: Cloud Logging filter expression to route logs to this table<br/>- description: Table description (optional) | <pre>map(object({<br/>    schema            = string<br/>    partition_field   = string<br/>    clustering_fields = optional(list(string), null)<br/>    log_filter        = string<br/>    description       = optional(string, "")<br/>  }))</pre> | n/a | yes |
| <a name="input_team"></a> [team](#input\_team) | Team label for resources | `string` | `null` | no |
| <a name="input_use_partitioned_tables"></a> [use\_partitioned\_tables](#input\_use\_partitioned\_tables) | Whether to use partitioned tables in log sink destinations | `bool` | `true` | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_dataset_id"></a> [dataset\_id](#output\_dataset\_id) | The ID of the BigQuery dataset |
| <a name="output_sink_names"></a> [sink\_names](#output\_sink\_names) | Map of table names to their log sink names |
| <a name="output_table_ids"></a> [table\_ids](#output\_table\_ids) | Map of table names to their IDs |
<!-- END_TF_DOCS -->
