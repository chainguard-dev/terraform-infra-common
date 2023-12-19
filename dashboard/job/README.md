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
| <a name="module_cpu_utilization"></a> [cpu\_utilization](#module\_cpu\_utilization) | ../widgets/xy | n/a |
| <a name="module_logs"></a> [logs](#module\_logs) | ../widgets/logs | n/a |
| <a name="module_memory_utilization"></a> [memory\_utilization](#module\_memory\_utilization) | ../widgets/xy | n/a |
| <a name="module_received_bytes"></a> [received\_bytes](#module\_received\_bytes) | ../widgets/xy | n/a |
| <a name="module_sent_bytes"></a> [sent\_bytes](#module\_sent\_bytes) | ../widgets/xy | n/a |
| <a name="module_startup_latency"></a> [startup\_latency](#module\_startup\_latency) | ../widgets/xy | n/a |

## Resources

| Name | Type |
|------|------|
| [google_monitoring_dashboard.dashboard](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/monitoring_dashboard) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_job_name"></a> [job\_name](#input\_job\_name) | Name of the job(s) to monitor | `string` | n/a | yes |

## Outputs

No outputs.
<!-- END_TF_DOCS -->
