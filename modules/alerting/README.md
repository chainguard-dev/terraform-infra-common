<!-- BEGIN_TF_DOCS -->
## Requirements

No requirements.

## Providers

| Name | Version |
|------|---------|
| <a name="provider_google"></a> [google](#provider\_google) | n/a |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [google_logging_metric.cloud-run-failed-req](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/logging_metric) | resource |
| [google_logging_metric.cloud-run-scaling-failure](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/logging_metric) | resource |
| [google_logging_metric.cloudrun_timeout](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/logging_metric) | resource |
| [google_logging_metric.dockerhub_ratelimit](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/logging_metric) | resource |
| [google_logging_metric.github_ratelimit](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/logging_metric) | resource |
| [google_logging_metric.r2_same_ratelimit](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/logging_metric) | resource |
| [google_monitoring_alert_policy.bad-rollout](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/monitoring_alert_policy) | resource |
| [google_monitoring_alert_policy.cloud-run-failed-req](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/monitoring_alert_policy) | resource |
| [google_monitoring_alert_policy.cloud-run-scaling-failure](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/monitoring_alert_policy) | resource |
| [google_monitoring_alert_policy.cloudrun_timeout](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/monitoring_alert_policy) | resource |
| [google_monitoring_alert_policy.fatal](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/monitoring_alert_policy) | resource |
| [google_monitoring_alert_policy.oom](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/monitoring_alert_policy) | resource |
| [google_monitoring_alert_policy.panic](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/monitoring_alert_policy) | resource |
| [google_monitoring_alert_policy.panic-stacktrace](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/monitoring_alert_policy) | resource |
| [google_monitoring_alert_policy.pubsub_dead_letter_queue_messages](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/monitoring_alert_policy) | resource |
| [google_monitoring_alert_policy.service_failure_rate_eventing](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/monitoring_alert_policy) | resource |
| [google_monitoring_alert_policy.service_failure_rate_non_eventing](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/monitoring_alert_policy) | resource |
| [google_monitoring_alert_policy.signal](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/monitoring_alert_policy) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_dlq_filter"></a> [dlq\_filter](#input\_dlq\_filter) | additional filter to apply to dlq alert policy | `string` | `""` | no |
| <a name="input_enable_scaling_alerts"></a> [enable\_scaling\_alerts](#input\_enable\_scaling\_alerts) | Whether to enable scaling alerts.<br/>  When logs appear with<br/>    "The request was aborted because there was no available instance." or<br/>    "The request failed because either the HTTP response was malformed or connection to the instance had an error." | `bool` | `false` | no |
| <a name="input_failed_req_filter"></a> [failed\_req\_filter](#input\_failed\_req\_filter) | additional filter to apply to failed request alert policy | `string` | `""` | no |
| <a name="input_failure_rate_duration"></a> [failure\_rate\_duration](#input\_failure\_rate\_duration) | duration for condition to be active before alerting | `number` | `120` | no |
| <a name="input_failure_rate_exclude_services"></a> [failure\_rate\_exclude\_services](#input\_failure\_rate\_exclude\_services) | List of service names to exclude from the 5xx failure rate alert | `list(string)` | `[]` | no |
| <a name="input_failure_rate_ratio_threshold"></a> [failure\_rate\_ratio\_threshold](#input\_failure\_rate\_ratio\_threshold) | ratio threshold to alert for cloud run server failure rate. | `number` | `0.2` | no |
| <a name="input_global_only_alerts"></a> [global\_only\_alerts](#input\_global\_only\_alerts) | only enable global alerts. when true, only create alerts that are global. | `bool` | `false` | no |
| <a name="input_notification_channels"></a> [notification\_channels](#input\_notification\_channels) | List of notification channels to alert. | `list(string)` | `[]` | no |
| <a name="input_notification_channels_email"></a> [notification\_channels\_email](#input\_notification\_channels\_email) | Email notification channel. | `list(string)` | `[]` | no |
| <a name="input_notification_channels_pagerduty"></a> [notification\_channels\_pagerduty](#input\_notification\_channels\_pagerduty) | Email notification channel. | `list(string)` | `[]` | no |
| <a name="input_notification_channels_pubsub"></a> [notification\_channels\_pubsub](#input\_notification\_channels\_pubsub) | Pubsub notification channel. | `list(string)` | `[]` | no |
| <a name="input_notification_channels_slack"></a> [notification\_channels\_slack](#input\_notification\_channels\_slack) | Slack notification channel. | `list(string)` | `[]` | no |
| <a name="input_oom_filter"></a> [oom\_filter](#input\_oom\_filter) | additional filter to apply to oom alert policy | `string` | `""` | no |
| <a name="input_panic_filter"></a> [panic\_filter](#input\_panic\_filter) | additional filter to apply to panic alert policy | `string` | `""` | no |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | n/a | `string` | n/a | yes |
| <a name="input_scaling_issue_filter"></a> [scaling\_issue\_filter](#input\_scaling\_issue\_filter) | additional filter to apply to scaling issue alert policy | `string` | `""` | no |
| <a name="input_signal_filter"></a> [signal\_filter](#input\_signal\_filter) | additional filter to apply to signal alert policy | `string` | `""` | no |
| <a name="input_squad"></a> [squad](#input\_squad) | squad to filter on if non-empty | `string` | `""` | no |
| <a name="input_timeout_filter"></a> [timeout\_filter](#input\_timeout\_filter) | additional filter to apply to timeout alert policy | `string` | `""` | no |

## Outputs

No outputs.
<!-- END_TF_DOCS -->
