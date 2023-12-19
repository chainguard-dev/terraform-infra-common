<!-- BEGIN_TF_DOCS -->
## Requirements

No requirements.

## Providers

| Name | Version |
|------|---------|
| <a name="provider_google"></a> [google](#provider\_google) | n/a |

## Modules

| Name | Source | Version |
|------|--------|---------|
| <a name="module_alert"></a> [alert](#module\_alert) | ../tiles/alert | n/a |
| <a name="module_cpu_utilization"></a> [cpu\_utilization](#module\_cpu\_utilization) | ../tiles/xy | n/a |
| <a name="module_logs"></a> [logs](#module\_logs) | ../tiles/logs | n/a |
| <a name="module_memory_utilization"></a> [memory\_utilization](#module\_memory\_utilization) | ../tiles/xy | n/a |
| <a name="module_received_bytes"></a> [received\_bytes](#module\_received\_bytes) | ../tiles/xy | n/a |
| <a name="module_sent_bytes"></a> [sent\_bytes](#module\_sent\_bytes) | ../tiles/xy | n/a |
| <a name="module_startup_latency"></a> [startup\_latency](#module\_startup\_latency) | ../tiles/xy | n/a |

## Resources

| Name | Type |
|------|------|
| [google_monitoring_dashboard.dashboard](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/monitoring_dashboard) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_alert_policies"></a> [alert\_policies](#input\_alert\_policies) | n/a | <pre>map(object({<br>    id = string<br>  }))</pre> | `{}` | no |
| <a name="input_job_name"></a> [job\_name](#input\_job\_name) | n/a | `string` | n/a | yes |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | n/a | `string` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_url"></a> [url](#output\_url) | n/a |
<!-- END_TF_DOCS -->
