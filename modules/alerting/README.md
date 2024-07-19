<!-- BEGIN_TF_DOCS -->

## Requirements

No requirements.

## Providers

| Name                                                      | Version |
| --------------------------------------------------------- | ------- |
| <a name="provider_google"></a> [google](#provider_google) | n/a     |

## Modules

No modules.

## Resources

| Name                                                                                                                                                 | Type     |
| ---------------------------------------------------------------------------------------------------------------------------------------------------- | -------- |
| [google_monitoring_alert_policy.bad-rollout](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/monitoring_alert_policy) | resource |
| [google_monitoring_alert_policy.fatal](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/monitoring_alert_policy)       | resource |
| [google_monitoring_alert_policy.oom](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/monitoring_alert_policy)         | resource |
| [google_monitoring_alert_policy.panic](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/monitoring_alert_policy)       | resource |

## Inputs

| Name                                                                                             | Description                                    | Type           | Default | Required |
| ------------------------------------------------------------------------------------------------ | ---------------------------------------------- | -------------- | ------- | :------: |
| <a name="input_notification_channels"></a> [notification_channels](#input_notification_channels) | List of notification channels to alert.        | `list(string)` | n/a     |   yes    |
| <a name="input_oom_filter"></a> [oom_filter](#input_oom_filter)                                  | additional filter to apply to oom alert policy | `string`       | n/a     |   yes    |
| <a name="input_project_id"></a> [project_id](#input_project_id)                                  | n/a                                            | `string`       | n/a     |   yes    |

## Outputs

No outputs.

<!-- END_TF_DOCS -->
