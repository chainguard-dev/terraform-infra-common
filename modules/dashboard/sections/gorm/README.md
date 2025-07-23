<!-- BEGIN_TF_DOCS -->
## Requirements

No requirements.

## Providers

No providers.

## Modules

| Name | Source | Version |
|------|--------|---------|
| <a name="module_collapsible"></a> [collapsible](#module\_collapsible) | ../collapsible | n/a |
| <a name="module_error_rate"></a> [error\_rate](#module\_error\_rate) | ../../widgets/percent | n/a |
| <a name="module_op_request_count"></a> [op\_request\_count](#module\_op\_request\_count) | ../../widgets/xy | n/a |
| <a name="module_open_connections"></a> [open\_connections](#module\_open\_connections) | ../../widgets/xy | n/a |
| <a name="module_request_errors"></a> [request\_errors](#module\_request\_errors) | ../../widgets/xy | n/a |
| <a name="module_table_request_count"></a> [table\_request\_count](#module\_table\_request\_count) | ../../widgets/xy | n/a |
| <a name="module_total_request_count"></a> [total\_request\_count](#module\_total\_request\_count) | ../../widgets/xy | n/a |
| <a name="module_width"></a> [width](#module\_width) | ../width | n/a |

## Resources

No resources.

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_collapsed"></a> [collapsed](#input\_collapsed) | n/a | `bool` | `false` | no |
| <a name="input_filter"></a> [filter](#input\_filter) | n/a | `list(string)` | n/a | yes |
| <a name="input_service_name"></a> [service\_name](#input\_service\_name) | n/a | `string` | n/a | yes |
| <a name="input_title"></a> [title](#input\_title) | n/a | `string` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_section"></a> [section](#output\_section) | n/a |
<!-- END_TF_DOCS -->
