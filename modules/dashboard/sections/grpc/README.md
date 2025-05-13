<!-- BEGIN_TF_DOCS -->
## Requirements

No requirements.

## Providers

No providers.

## Modules

| Name | Source | Version |
|------|--------|---------|
| <a name="module_collapsible"></a> [collapsible](#module\_collapsible) | ../collapsible | n/a |
| <a name="module_failure_rate"></a> [failure\_rate](#module\_failure\_rate) | ../../widgets/percent | n/a |
| <a name="module_incoming_latency"></a> [incoming\_latency](#module\_incoming\_latency) | ../../widgets/latency | n/a |
| <a name="module_outbound_latency"></a> [outbound\_latency](#module\_outbound\_latency) | ../../widgets/latency | n/a |
| <a name="module_outbound_request_count"></a> [outbound\_request\_count](#module\_outbound\_request\_count) | ../../widgets/xy | n/a |
| <a name="module_request_count"></a> [request\_count](#module\_request\_count) | ../../widgets/xy | n/a |
| <a name="module_width"></a> [width](#module\_width) | ../width | n/a |

## Resources

No resources.

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_collapsed"></a> [collapsed](#input\_collapsed) | n/a | `bool` | `false` | no |
| <a name="input_filter"></a> [filter](#input\_filter) | n/a | `list(string)` | n/a | yes |
| <a name="input_grpc_non_error_codes"></a> [grpc\_non\_error\_codes](#input\_grpc\_non\_error\_codes) | List of grpc codes to not counted as error, case-sensitive. | `list(string)` | <pre>[<br/>  "OK",<br/>  "Aborted",<br/>  "AlreadyExists",<br/>  "Canceled",<br/>  "NotFound"<br/>]</pre> | no |
| <a name="input_service_name"></a> [service\_name](#input\_service\_name) | n/a | `string` | n/a | yes |
| <a name="input_title"></a> [title](#input\_title) | n/a | `string` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_section"></a> [section](#output\_section) | n/a |
<!-- END_TF_DOCS -->
