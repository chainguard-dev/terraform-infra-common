<!-- BEGIN_TF_DOCS -->
## Requirements

No requirements.

## Providers

No providers.

## Modules

| Name | Source | Version |
|------|--------|---------|
| <a name="module_collapsible"></a> [collapsible](#module\_collapsible) | ../collapsible | n/a |
| <a name="module_limit"></a> [limit](#module\_limit) | ../../widgets/xy | n/a |
| <a name="module_remaining"></a> [remaining](#module\_remaining) | ../../widgets/xy | n/a |
| <a name="module_reset"></a> [reset](#module\_reset) | ../../widgets/xy | n/a |
| <a name="module_width"></a> [width](#module\_width) | ../width | n/a |

## Resources

No resources.

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_cloudrun_name"></a> [cloudrun\_name](#input\_cloudrun\_name) | n/a | `string` | n/a | yes |
| <a name="input_collapsed"></a> [collapsed](#input\_collapsed) | n/a | `bool` | `false` | no |
| <a name="input_filter"></a> [filter](#input\_filter) | n/a | `list(string)` | n/a | yes |
| <a name="input_title"></a> [title](#input\_title) | n/a | `string` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_section"></a> [section](#output\_section) | n/a |
<!-- END_TF_DOCS -->
