<!-- BEGIN_TF_DOCS -->
## Requirements

No requirements.

## Providers

| Name | Version |
|------|---------|
| <a name="provider_google"></a> [google](#provider\_google) | n/a |

## Modules

| Name | Source | Version |
|------|--------|---------|
| <a name="module_oldest-unacked"></a> [oldest-unacked](#module\_oldest-unacked) | ../tiles/xy | n/a |
| <a name="module_push-latency"></a> [push-latency](#module\_push-latency) | ../tiles/latency | n/a |
| <a name="module_received-events"></a> [received-events](#module\_received-events) | ../tiles/xy | n/a |
| <a name="module_undelivered"></a> [undelivered](#module\_undelivered) | ../tiles/xy | n/a |

## Resources

| Name | Type |
|------|------|
| [google_monitoring_dashboard.dashboard](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/monitoring_dashboard) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_alert_policies"></a> [alert\_policies](#input\_alert\_policies) | n/a | <pre>map(object({<br>    id = string<br>  }))</pre> | `{}` | no |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | n/a | `string` | n/a | yes |
| <a name="input_topic_prefix"></a> [topic\_prefix](#input\_topic\_prefix) | n/a | `string` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_url"></a> [url](#output\_url) | n/a |
<!-- END_TF_DOCS -->