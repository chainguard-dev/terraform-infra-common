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
| <a name="input_band"></a> [band](#input\_band) | n/a | `number` | `99` | no |
| <a name="input_filter"></a> [filter](#input\_filter) | n/a | `list(string)` | n/a | yes |
| <a name="input_group_by_fields"></a> [group\_by\_fields](#input\_group\_by\_fields) | n/a | `list` | `[]` | no |
| <a name="input_title"></a> [title](#input\_title) | n/a | `string` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_widget"></a> [widget](#output\_widget) | https://cloud.google.com/monitoring/api/ref_v3/rest/v1/projects.dashboards#XyChart |
<!-- END_TF_DOCS -->
