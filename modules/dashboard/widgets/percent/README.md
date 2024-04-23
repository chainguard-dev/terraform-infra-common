<!-- BEGIN_TF_DOCS -->
## Requirements

No requirements.

## Providers

No providers.

## Modules

| Name | Source | Version |
|------|--------|---------|
| <a name="module_plot"></a> [plot](#module\_plot) | ../xy-ratio | n/a |

## Resources

No resources.

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_align"></a> [align](#input\_align) | n/a | `string` | `"ALIGN_RATE"` | no |
| <a name="input_alignment_period"></a> [alignment\_period](#input\_alignment\_period) | n/a | `string` | `"60s"` | no |
| <a name="input_common_filter"></a> [common\_filter](#input\_common\_filter) | n/a | `list(string)` | n/a | yes |
| <a name="input_group_by_fields"></a> [group\_by\_fields](#input\_group\_by\_fields) | n/a | `list` | `[]` | no |
| <a name="input_legend"></a> [legend](#input\_legend) | n/a | `string` | `""` | no |
| <a name="input_numerator_additional_filter"></a> [numerator\_additional\_filter](#input\_numerator\_additional\_filter) | n/a | `list(string)` | n/a | yes |
| <a name="input_plot_type"></a> [plot\_type](#input\_plot\_type) | n/a | `string` | `"LINE"` | no |
| <a name="input_reduce"></a> [reduce](#input\_reduce) | n/a | `string` | `"REDUCE_SUM"` | no |
| <a name="input_thresholds"></a> [thresholds](#input\_thresholds) | n/a | `list` | `[]` | no |
| <a name="input_title"></a> [title](#input\_title) | n/a | `string` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_widget"></a> [widget](#output\_widget) | n/a |
<!-- END_TF_DOCS -->
