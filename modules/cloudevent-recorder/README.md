# `cloudevent-recorder`

This module provisions a regionalized cloudevents sink that consumes events of
particular types from each of the regional brokers, and writes them into a
regional GCS bucket where a periodic BigQuery Data-Transfer Service Job will
pull events from into a BigQuery table schematized for that event type. The
intended usage of this module for publishing events is something like this:

```hcl
// Create a network with several regional subnets
module "networking" {
  source = "chainguard-dev/common/infra//modules/networking"

  name       = "my-networking"
  project_id = var.project_id
  regions    = [...]
}

// Create the Broker abstraction.
module "cloudevent-broker" {
  source = "chainguard-dev/common/infra//modules/cloudevent-broker"

  name       = "my-broker"
  project_id = var.project_id
  regions    = module.networking.regional-networks
}

// Record cloudevents of type com.example.foo and com.example.bar
module "foo-emits-events" {
  source = "chainguard-dev/common/infra//modules/cloudevent-recorder"

  name       = "my-recorder"
  project_id = var.project_id
  regions    = module.networking.regional-networks
  broker     = module.cloudevent-broker.broker

  retention-period = 30 // keep around 30 days worth of event data

  provisioner = "user:sally@chainguard.dev"

  types = {
    "com.example.foo": {
      schema = file("${path.module}/foo.schema.json")
    }
    "com.example.bar": {
      schema = file("${path.module}/bar.schema.json")
    }
  }
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

| Name | Source | Version |
|------|--------|---------|
| <a name="module_audit-import-serviceaccount"></a> [audit-import-serviceaccount](#module\_audit-import-serviceaccount) | ../audit-serviceaccount | n/a |
| <a name="module_recorder-dashboard"></a> [recorder-dashboard](#module\_recorder-dashboard) | ../dashboard/cloudevent-receiver | n/a |
| <a name="module_this"></a> [this](#module\_this) | ../regional-go-service | n/a |
| <a name="module_triggers"></a> [triggers](#module\_triggers) | ../cloudevent-trigger | n/a |

## Resources

| Name | Type |
|------|------|
| [google_bigquery_data_transfer_config.import-job](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/bigquery_data_transfer_config) | resource |
| [google_bigquery_dataset.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/bigquery_dataset) | resource |
| [google_bigquery_table.types](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/bigquery_table) | resource |
| [google_bigquery_table_iam_binding.import-writes-to-tables](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/bigquery_table_iam_binding) | resource |
| [google_monitoring_alert_policy.bq_dts](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/monitoring_alert_policy) | resource |
| [google_service_account.import-identity](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/service_account) | resource |
| [google_service_account.recorder](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/service_account) | resource |
| [google_service_account_iam_binding.bq-dts-assumes-import-identity](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/service_account_iam_binding) | resource |
| [google_service_account_iam_binding.provisioner-acts-as-import-identity](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/service_account_iam_binding) | resource |
| [google_storage_bucket.recorder](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/storage_bucket) | resource |
| [google_storage_bucket_iam_binding.import-reads-from-gcs-buckets](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/storage_bucket_iam_binding) | resource |
| [google_storage_bucket_iam_binding.recorder-writes-to-gcs-buckets](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/storage_bucket_iam_binding) | resource |
| [random_id.suffix](https://registry.terraform.io/providers/hashicorp/random/latest/docs/resources/id) | resource |
| [random_id.trigger-suffix](https://registry.terraform.io/providers/hashicorp/random/latest/docs/resources/id) | resource |
| [google_project.project](https://registry.terraform.io/providers/hashicorp/google/latest/docs/data-sources/project) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_broker"></a> [broker](#input\_broker) | A map from each of the input region names to the name of the Broker topic in that region. | `map(string)` | n/a | yes |
| <a name="input_deletion_protection"></a> [deletion\_protection](#input\_deletion\_protection) | Whether to enable deletion protection on data resources. | `bool` | `true` | no |
| <a name="input_location"></a> [location](#input\_location) | The location to create the BigQuery dataset in, and in which to run the data transfer jobs from GCS. | `string` | `"US"` | no |
| <a name="input_name"></a> [name](#input\_name) | n/a | `string` | n/a | yes |
| <a name="input_notification_channels"></a> [notification\_channels](#input\_notification\_channels) | List of notification channels to alert (for service-level issues). | `list(string)` | `[]` | no |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | n/a | `string` | n/a | yes |
| <a name="input_provisioner"></a> [provisioner](#input\_provisioner) | The identity as which this module will be applied (so it may be granted permission to 'act as' the DTS service account).  This should be in the form expected by an IAM subject (e.g. user:sally@example.com) | `string` | n/a | yes |
| <a name="input_regions"></a> [regions](#input\_regions) | A map from region names to a network and subnetwork.  A recorder service and cloud storage bucket (into which the service writes events) will be created in each region. | <pre>map(object({<br>    network = string<br>    subnet  = string<br>  }))</pre> | n/a | yes |
| <a name="input_retention-period"></a> [retention-period](#input\_retention-period) | The number of days to retain data in BigQuery. | `number` | n/a | yes |
| <a name="input_types"></a> [types](#input\_types) | A map from cloudevent types to the BigQuery schema associated with them, as well as an alert threshold and a list of notification channels (for subscription-level issues). | <pre>map(object({<br>    schema                = string<br>    alert_threshold       = optional(number, 50000)<br>    notification_channels = optional(list(string), [])<br>  }))</pre> | n/a | yes |

## Outputs

No outputs.
<!-- END_TF_DOCS -->
