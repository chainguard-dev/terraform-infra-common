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
| [google_monitoring_alert_policy.slo_burn_rate_multi_region](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/monitoring_alert_policy) | resource |
| [google_monitoring_alert_policy.slo_burn_rate_per_region](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/monitoring_alert_policy) | resource |
| [google_monitoring_service.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/monitoring_service) | resource |
| [google_monitoring_slo.availability](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/monitoring_slo) | resource |
| [google_monitoring_slo.availability_per_region](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/monitoring_slo) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_notification_channels"></a> [notification\_channels](#input\_notification\_channels) | List of notification channels to alert. | `list(string)` | n/a | yes |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | GCP project ID | `string` | n/a | yes |
| <a name="input_regions"></a> [regions](#input\_regions) | A list of regions that the cloudrun service is deployed in. | `list(string)` | n/a | yes |
| <a name="input_service_name"></a> [service\_name](#input\_service\_name) | Name of service to setup SLO for. | `string` | n/a | yes |
| <a name="input_service_type"></a> [service\_type](#input\_service\_type) | Type of service to setup SLO for. | `string` | `"CLOUD_RUN"` | no |
| <a name="input_slo"></a> [slo](#input\_slo) | Configuration for setting up SLO | <pre>object({<br/>    enable          = optional(bool, false)<br/>    enable_alerting = optional(bool, false)<br/>    availability = optional(object(<br/>      {<br/>        multi_region_goal = optional(number, 0.999)<br/>        per_region_goal   = optional(number, 0.999)<br/>      }<br/>    ), null)<br/>  })</pre> | `{}` | no |

## Outputs

No outputs.
<!-- END_TF_DOCS -->
