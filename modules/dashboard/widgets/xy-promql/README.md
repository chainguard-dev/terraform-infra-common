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
| <a name="input_plot_type"></a> [plot\_type](#input\_plot\_type) | Plot type for the chart (LINE, AREA, STACKED\_AREA, STACKED\_BAR) | `string` | `"LINE"` | no |
| <a name="input_promql_query"></a> [promql\_query](#input\_promql\_query) | PromQL query for the time series data | `string` | n/a | yes |
| <a name="input_thresholds"></a> [thresholds](#input\_thresholds) | List of threshold values to display on the chart | `list(number)` | `[]` | no |
| <a name="input_timeshift_duration"></a> [timeshift\_duration](#input\_timeshift\_duration) | Duration to timeshift the data | `string` | `"0s"` | no |
| <a name="input_title"></a> [title](#input\_title) | Title of the XY chart widget | `string` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_widget"></a> [widget](#output\_widget) | n/a |
<!-- END_TF_DOCS -->
