# `service-health`

This module sends Google Cloud incidents that affect a project to Slack, email,
Pub/Sub, or any other Cloud Monitoring notification channel, using
[Personalized Service Health](https://docs.cloud.google.com/service-health/docs/overview).

Unlike the public [status.cloud.google.com](https://status.cloud.google.com)
feed, Personalized Service Health only reports incidents relevant to the
project. Each time an incident is created or updated it writes an `EventLog`
entry to Cloud Logging. This module enables `servicehealth.googleapis.com` and
creates a log-match alert policy on those entries.

Service Health is project-scoped: instantiate the module once per project you
want covered.

```hcl
# Slack notification channels cannot be created through Terraform
# (https://github.com/hashicorp/terraform-provider-google/issues/11346),
# so create them in the console and look them up.
data "google_monitoring_notification_channel" "statuspage" {
  display_name = "Slack statuspage"
}

module "service-health" {
  source = "chainguard-dev/common/infra//modules/service-health"

  project_id = var.project_id
  name       = "enforce.dev"

  notification_channels_slack = [
    data.google_monitoring_notification_channel.statuspage.name,
  ]
  notification_channels_email  = [google_monitoring_notification_channel.oncall_email.name]
  notification_channels_pubsub = [google_monitoring_notification_channel.incidents_pubsub.name]

  # Optional: only alert on what this environment runs.
  products  = ["Cloud Run", "Google Kubernetes Engine", "Cloud SQL"]
  locations = ["us-central1", "global"]
}
```

The notification rate limit applies to the whole policy, not per incident: with
the default `300s`, an update to a second incident within five minutes of the
first notification is dropped.

<!-- BEGIN_TF_DOCS -->
## Requirements

No requirements.

## Providers

| Name | Version |
| ---- | ------- |
| <a name="provider_google"></a> [google](#provider\_google) | n/a |

## Modules

No modules.

## Resources

| Name | Type |
| ---- | ---- |
| [google_monitoring_alert_policy.incident](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/monitoring_alert_policy) | resource |
| [google_project_service.servicehealth](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/project_service) | resource |

## Inputs

| Name | Description | Type | Default | Required |
| ---- | ----------- | ---- | ------- | :------: |
| <a name="input_auto_close"></a> [auto\_close](#input\_auto\_close) | How long a Monitoring incident opened by the policy stays open before it auto-closes. Service Health posts its own RESOLVED update, so this only controls Monitoring incident bookkeeping. | `string` | `"1800s"` | no |
| <a name="input_enable_api"></a> [enable\_api](#input\_enable\_api) | Whether to enable servicehealth.googleapis.com in the project. Set to false if the API is enabled elsewhere. | `bool` | `true` | no |
| <a name="input_locations"></a> [locations](#input\_locations) | Only alert on incidents impacting at least one of these locations, matched as a substring of the event's impacted locations (e.g. "us-central1", "global"). Empty alerts on every location. | `list(string)` | `[]` | no |
| <a name="input_name"></a> [name](#input\_name) | Optional environment name (e.g. enforce.dev) prefixed to the notification subject, so several environments can share a channel. | `string` | `""` | no |
| <a name="input_notification_channels"></a> [notification\_channels](#input\_notification\_channels) | Any other notification channels (e.g. webhook, PagerDuty, incident.io), as projects/<project>/notificationChannels/<id>. | `list(string)` | `[]` | no |
| <a name="input_notification_channels_email"></a> [notification\_channels\_email](#input\_notification\_channels\_email) | Email notification channels to notify, as projects/<project>/notificationChannels/<id>. | `list(string)` | `[]` | no |
| <a name="input_notification_channels_pubsub"></a> [notification\_channels\_pubsub](#input\_notification\_channels\_pubsub) | Pub/Sub notification channels to notify, as projects/<project>/notificationChannels/<id>. Useful for routing incidents to a custom formatter or another system. | `list(string)` | `[]` | no |
| <a name="input_notification_channels_slack"></a> [notification\_channels\_slack](#input\_notification\_channels\_slack) | Slack notification channels to notify, as projects/<project>/notificationChannels/<id>. Slack channels cannot be created through Terraform, so create them in the console and look them up with the google\_monitoring\_notification\_channel data source. | `list(string)` | `[]` | no |
| <a name="input_notification_rate_limit"></a> [notification\_rate\_limit](#input\_notification\_rate\_limit) | Minimum time between notifications from the alert policy. Log-match policies notify per matching log entry, and the limit applies across all incidents, so a large value can delay updates for a second, concurrent incident. | `string` | `"300s"` | no |
| <a name="input_products"></a> [products](#input\_products) | Only alert on incidents impacting at least one of these products, matched as a substring of the event's impacted products (e.g. "Cloud Run", "Google Kubernetes Engine"). Empty alerts on every product. | `list(string)` | `[]` | no |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | The GCP project whose Personalized Service Health events are alerted on. Service Health is project-scoped: only incidents relevant to this project are reported. | `string` | n/a | yes |
| <a name="input_severity"></a> [severity](#input\_severity) | Severity of the alert policy: CRITICAL, ERROR, or WARNING. | `string` | `"WARNING"` | no |
| <a name="input_team"></a> [team](#input\_team) | Team label for the alert policy. | `string` | `null` | no |

## Outputs

| Name | Description |
| ---- | ----------- |
| <a name="output_alert_policy_name"></a> [alert\_policy\_name](#output\_alert\_policy\_name) | The resource name of the Service Health alert policy (projects/<project>/alertPolicies/<id>). |
| <a name="output_filter"></a> [filter](#output\_filter) | The Cloud Logging filter matched by the alert policy, for reuse in log views or sinks. |
<!-- END_TF_DOCS -->
