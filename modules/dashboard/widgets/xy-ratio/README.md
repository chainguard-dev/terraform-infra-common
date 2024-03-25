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
| <a name="input_denominator_align"></a> [denominator\_align](#input\_denominator\_align) | n/a | `string` | `"ALIGN_RATE"` | no |
| <a name="input_denominator_filter"></a> [denominator\_filter](#input\_denominator\_filter) | n/a | `list(string)` | n/a | yes |
| <a name="input_denominator_group_by_fields"></a> [denominator\_group\_by\_fields](#input\_denominator\_group\_by\_fields) | n/a | `list` | `[]` | no |
| <a name="input_denominator_reduce"></a> [denominator\_reduce](#input\_denominator\_reduce) | n/a | `string` | `"REDUCE_SUM"` | no |
| <a name="input_legend"></a> [legend](#input\_legend) | n/a | `string` | `""` | no |
| <a name="input_numerator_align"></a> [numerator\_align](#input\_numerator\_align) | n/a | `string` | `"ALIGN_RATE"` | no |
| <a name="input_numerator_filter"></a> [numerator\_filter](#input\_numerator\_filter) | n/a | `list(string)` | n/a | yes |
| <a name="input_numerator_group_by_fields"></a> [numerator\_group\_by\_fields](#input\_numerator\_group\_by\_fields) | n/a | `list` | `[]` | no |
| <a name="input_numerator_reduce"></a> [numerator\_reduce](#input\_numerator\_reduce) | n/a | `string` | `"REDUCE_SUM"` | no |
| <a name="input_plot_type"></a> [plot\_type](#input\_plot\_type) | n/a | `string` | `"LINE"` | no |
| <a name="input_thresholds"></a> [thresholds](#input\_thresholds) | n/a | `list` | `[]` | no |
| <a name="input_title"></a> [title](#input\_title) | n/a | `string` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_widget"></a> [widget](#output\_widget) | https://cloud.google.com/monitoring/api/ref_v3/rest/v1/projects.dashboards#XyChart |
<!-- END_TF_DOCS -->