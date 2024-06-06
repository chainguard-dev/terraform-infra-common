<!-- BEGIN_TF_DOCS -->
## Requirements

No requirements.

## Providers

No providers.

## Modules

No modules.

## Resources

No resources.

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_alignment_period"></a> [alignment\_period](#input\_alignment\_period) | n/a | `string` | `"60s"` | no |
| <a name="input_filter"></a> [filter](#input\_filter) | n/a | `list(string)` | n/a | yes |
| <a name="input_group_by_fields"></a> [group\_by\_fields](#input\_group\_by\_fields) | n/a | `list` | `[]` | no |
| <a name="input_plot_type"></a> [plot\_type](#input\_plot\_type) | n/a | `string` | `"LINE"` | no |
| <a name="input_primary_align"></a> [primary\_align](#input\_primary\_align) | n/a | `string` | `"ALIGN_RATE"` | no |
| <a name="input_primary_reduce"></a> [primary\_reduce](#input\_primary\_reduce) | n/a | `string` | `"REDUCE_NONE"` | no |
| <a name="input_secondary_align"></a> [secondary\_align](#input\_secondary\_align) | n/a | `string` | `"ALIGN_NONE"` | no |
| <a name="input_secondary_reduce"></a> [secondary\_reduce](#input\_secondary\_reduce) | n/a | `string` | `"REDUCE_NONE"` | no |
| <a name="input_title"></a> [title](#input\_title) | n/a | `string` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_widget"></a> [widget](#output\_widget) | https://cloud.google.com/monitoring/api/ref_v3/rest/v1/projects.dashboards#XyChart |
<!-- END_TF_DOCS -->
