# BigQuery Log Sink Module

Terraform module for creating BigQuery datasets with Cloud Logging sinks for structured log ingestion.

## Features

- Creates BigQuery dataset with configurable expiration
- Sets up Cloud Logging sinks to route logs to BigQuery
- Tables are auto-created by Cloud Logging based on log names
- Optional monitoring alerts for log ingestion health

## How It Works

Cloud Logging automatically creates tables in BigQuery based on the log name. For example:
- `run.googleapis.com/stderr` logs create a table named `run_googleapis_com_stderr`
- `run.googleapis.com/stdout` logs create a table named `run_googleapis_com_stdout`

See: https://cloud.google.com/logging/docs/export/bigquery

## Usage

```hcl
module "bigquery_log_sink" {
  source = "../../terraform/public-modules/modules/bigquery-logsink"

  project_id = var.project_id
  name       = "my_service"
  location   = "US"

  # 30 days in milliseconds - partitions older than this will be deleted
  default_partition_expiration_ms = 2592000000

  team    = "platform"
  product = "logging"

  sinks = {
    error_logs = {
      description = "Error logs from Cloud Run services"
      log_filter  = "resource.type=\"cloud_run_revision\" AND severity>=ERROR"
    }

    audit_logs = {
      description = "Audit logs for compliance"
      log_filter  = "resource.type=\"cloud_run_revision\" AND jsonPayload.log_type=\"audit\""
    }
  }

  enable_monitoring = false
}
```

## Variables

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| project_id | The GCP project ID | `string` | n/a | yes |
| name | Base name for resources | `string` | n/a | yes |
| location | BigQuery dataset location | `string` | `"US"` | no |
| default_partition_expiration_ms | Partition expiration in milliseconds | `number` | `2592000000` (30 days) | no |
| sinks | Map of log sinks to create | `map(object)` | n/a | yes |
| use_partitioned_tables | Use partitioned tables (required for partition expiration) | `bool` | `true` | no |
| team | Team label | `string` | `null` | no |
| product | Product label | `string` | `null` | no |
| enable_monitoring | Enable monitoring alerts | `bool` | `false` | no |

See `variables.tf` for all available configuration options.

## Outputs

| Name | Description |
|------|-------------|
| dataset_id | BigQuery dataset ID |
| sink_names | Map of sink keys to log sink names |
| sink_writer_identities | Map of sink keys to writer service accounts |

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
| [google_logging_project_sink.sinks](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/logging_project_sink) | resource |
| [google_monitoring_alert_policy.log_ingestion](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/monitoring_alert_policy) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_alert_auto_close_days"></a> [alert\_auto\_close\_days](#input\_alert\_auto\_close\_days) | Days after which to auto-close resolved alerts | `number` | `1` | no |
| <a name="input_alert_threshold_minutes"></a> [alert\_threshold\_minutes](#input\_alert\_threshold\_minutes) | Minutes without log ingestion before triggering alert | `number` | `180` | no |
| <a name="input_dataset_description"></a> [dataset\_description](#input\_dataset\_description) | Description of the BigQuery dataset | `string` | `""` | no |
| <a name="input_delete_contents_on_destroy"></a> [delete\_contents\_on\_destroy](#input\_delete\_contents\_on\_destroy) | Whether to delete dataset contents when destroying the dataset | `bool` | `false` | no |
| <a name="input_enable_monitoring"></a> [enable\_monitoring](#input\_enable\_monitoring) | Enable monitoring alert policies for log ingestion | `bool` | `false` | no |
| <a name="input_labels"></a> [labels](#input\_labels) | Additional labels to apply to resources | `map(string)` | `{}` | no |
| <a name="input_location"></a> [location](#input\_location) | BigQuery dataset location | `string` | `"US"` | no |
| <a name="input_name"></a> [name](#input\_name) | Base name for the BigQuery resources | `string` | n/a | yes |
| <a name="input_notification_channels"></a> [notification\_channels](#input\_notification\_channels) | List of notification channel IDs for alerts | `list(string)` | `[]` | no |
| <a name="input_product"></a> [product](#input\_product) | Product label for resources | `string` | `null` | no |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | The GCP project ID where resources will be created | `string` | n/a | yes |
| <a name="input_retention_days"></a> [retention\_days](#input\_retention\_days) | The number of days to retain data in BigQuery. Partitions older than this will be automatically deleted. Only applies when use\_partitioned\_tables is true. | `number` | `30` | no |
| <a name="input_sinks"></a> [sinks](#input\_sinks) | Map of log sinks to create. Each key is the sink name suffix, and the value is an object with:<br/>- log\_filter: Cloud Logging filter expression to route logs<br/>- description: Sink description (optional)<br/><br/>Note: Tables are auto-created by Cloud Logging based on log names.<br/>See: https://cloud.google.com/logging/docs/export/bigquery | <pre>map(object({<br/>    log_filter  = string<br/>    description = optional(string, "")<br/>  }))</pre> | n/a | yes |
| <a name="input_team"></a> [team](#input\_team) | Team label for resources | `string` | `null` | no |
| <a name="input_use_partitioned_tables"></a> [use\_partitioned\_tables](#input\_use\_partitioned\_tables) | Whether to use partitioned tables in log sink destinations. Must be true for partition expiration to work. | `bool` | `true` | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_dataset_id"></a> [dataset\_id](#output\_dataset\_id) | The ID of the BigQuery dataset |
| <a name="output_sink_names"></a> [sink\_names](#output\_sink\_names) | Map of sink keys to their log sink names |
| <a name="output_sink_writer_identities"></a> [sink\_writer\_identities](#output\_sink\_writer\_identities) | Map of sink keys to their writer identity service accounts |
<!-- END_TF_DOCS -->
