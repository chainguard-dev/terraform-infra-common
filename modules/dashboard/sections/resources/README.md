<!-- BEGIN_TF_DOCS -->
## Requirements

No requirements.

## Providers

No providers.

## Modules

| Name | Source | Version |
|------|--------|---------|
| <a name="module_collapsible"></a> [collapsible](#module\_collapsible) | ../collapsible | n/a |
| <a name="module_cpu_utilization"></a> [cpu\_utilization](#module\_cpu\_utilization) | ../../widgets/xy | n/a |
| <a name="module_disk_usage"></a> [disk\_usage](#module\_disk\_usage) | ../../widgets/xy | n/a |
| <a name="module_instance_count"></a> [instance\_count](#module\_instance\_count) | ../../widgets/xy | n/a |
| <a name="module_memory_utilization"></a> [memory\_utilization](#module\_memory\_utilization) | ../../widgets/xy | n/a |
| <a name="module_received_bytes"></a> [received\_bytes](#module\_received\_bytes) | ../../widgets/xy | n/a |
| <a name="module_sent_bytes"></a> [sent\_bytes](#module\_sent\_bytes) | ../../widgets/xy | n/a |
| <a name="module_startup_latency"></a> [startup\_latency](#module\_startup\_latency) | ../../widgets/xy | n/a |
| <a name="module_width"></a> [width](#module\_width) | ../width | n/a |

## Resources

No resources.

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_cloudrun_name"></a> [cloudrun\_name](#input\_cloudrun\_name) | n/a | `string` | n/a | yes |
| <a name="input_collapsed"></a> [collapsed](#input\_collapsed) | n/a | `bool` | `false` | no |
| <a name="input_filter"></a> [filter](#input\_filter) | n/a | `list(string)` | n/a | yes |
| <a name="input_notification_channels"></a> [notification\_channels](#input\_notification\_channels) | n/a | `list(string)` | n/a | yes |
| <a name="input_title"></a> [title](#input\_title) | n/a | `string` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_section"></a> [section](#output\_section) | n/a |
<!-- END_TF_DOCS -->
