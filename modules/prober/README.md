# `prober`

This module provisions a regionalized prober performing custom probes in Go.
The Go code is built and signed using the `ko` and `cosign` providers. The
simplest example service can be seen here:

```hcl
// Create a network with several regional subnets
module "networking" {
  source = "chainguard-dev/common/infra//modules/networking"

  name       = "my-networking"
  project_id = var.project_id
  regions    = [...]
}

module "foo-prober" {
  source  = "chainguard-dev/common/infra//modules/prober"

  name       = "my-prober"
  project_id = var.project_id
  regions    = module.networking.regional-networks

  # Needed if your service is outside of the regional networks.
  egress = "PRIVATE_RANGES_ONLY"

  # These are optional (e.g. for dev), but needed to put regional probers
  # behind GCLB for the uptime checks.
  domain   = var.domain
  dns_zone = google_dns_managed_zone.dns_zone.name

  # The service account as which to run the probers.
  service_account = google_service_account.foo-probes.email

  # Additional environment variables to pass the prober.
  env = {
    FOO = "bar"
  }

  # The source code for the custom prober.
  working_dir = path.module
  importpath  = "./cmd/my-prober"

  enable_alert          = true
  notification_channels = [ ... ]
}
```

The probes themselves can leverage our Go library to bootstrap, e.g.
```go
import (
	"context"
	"log"

	"github.com/chainguard-dev/terraform-infra-common/pkg/prober"
)

func main() {
	prober.Go(context.Background(), prober.Func(func(ctx context.Context) error {
		log.Print("Got a probe!")
		return nil
	}))
}
```

<!-- BEGIN_TF_DOCS -->
## Requirements

No requirements.

## Providers

| Name | Version |
|------|---------|
| <a name="provider_google"></a> [google](#provider\_google) | n/a |
| <a name="provider_random"></a> [random](#provider\_random) | n/a |

## Modules

| Name | Source | Version |
|------|--------|---------|
| <a name="module_gclb"></a> [gclb](#module\_gclb) | ../serverless-gclb | n/a |
| <a name="module_this"></a> [this](#module\_this) | ../regional-go-service | n/a |

## Resources

| Name | Type |
|------|------|
| [google_monitoring_alert_policy.slo_alert](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/monitoring_alert_policy) | resource |
| [google_monitoring_alert_policy.uptime_alert](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/monitoring_alert_policy) | resource |
| [google_monitoring_uptime_check_config.global_uptime_check](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/monitoring_uptime_check_config) | resource |
| [google_monitoring_uptime_check_config.regional_uptime_check](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/monitoring_uptime_check_config) | resource |
| [random_password.secret](https://registry.terraform.io/providers/hashicorp/random/latest/docs/resources/password) | resource |
| [google_cloud_run_v2_service.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/data-sources/cloud_run_v2_service) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_alert_description"></a> [alert\_description](#input\_alert\_description) | Alert documentation. Use this to link to playbooks or give additional context. | `string` | `"An uptime check has failed."` | no |
| <a name="input_base_image"></a> [base\_image](#input\_base\_image) | The base image to use for the prober. | `string` | `null` | no |
| <a name="input_cpu"></a> [cpu](#input\_cpu) | The CPU limit for the prober. | `string` | `"1000m"` | no |
| <a name="input_dns_zone"></a> [dns\_zone](#input\_dns\_zone) | The managed DNS zone in which to create prober record sets (required for multiple locations). | `string` | `""` | no |
| <a name="input_domain"></a> [domain](#input\_domain) | The domain of the environment to probe (required for multiple locations). | `string` | `""` | no |
| <a name="input_egress"></a> [egress](#input\_egress) | The level of egress the prober requires. | `string` | `"ALL_TRAFFIC"` | no |
| <a name="input_enable_alert"></a> [enable\_alert](#input\_enable\_alert) | If true, alert on failures. Outputs will return the alert ID for notification and dashboards. | `bool` | `false` | no |
| <a name="input_enable_profiler"></a> [enable\_profiler](#input\_enable\_profiler) | Enable cloud profiler. | `bool` | `false` | no |
| <a name="input_enable_slo_alert"></a> [enable\_slo\_alert](#input\_enable\_slo\_alert) | If true, alert service availability dropping below SLO threshold. Outputs will return the alert ID for notification and dashboards. | `bool` | `false` | no |
| <a name="input_env"></a> [env](#input\_env) | A map of custom environment variables (e.g. key=value) | `map` | `{}` | no |
| <a name="input_importpath"></a> [importpath](#input\_importpath) | The import path that contains the prober application. | `string` | n/a | yes |
| <a name="input_memory"></a> [memory](#input\_memory) | The memory limit for the prober. | `string` | `"512Mi"` | no |
| <a name="input_name"></a> [name](#input\_name) | Name to prefix to created resources. | `string` | n/a | yes |
| <a name="input_notification_channels"></a> [notification\_channels](#input\_notification\_channels) | A list of notification channels to send alerts to. | `list(string)` | n/a | yes |
| <a name="input_period"></a> [period](#input\_period) | The period for the prober in seconds. | `string` | `"300s"` | no |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | The project that will host the prober. | `string` | n/a | yes |
| <a name="input_regions"></a> [regions](#input\_regions) | A map from region names to a network and subnetwork.  A prober service will be created in each region. | <pre>map(object({<br>    network = string<br>    subnet  = string<br>  }))</pre> | n/a | yes |
| <a name="input_secret_env"></a> [secret\_env](#input\_secret\_env) | A map of secrets to mount as environment variables from Google Secrets Manager (e.g. secret\_key=secret\_name) | `map` | `{}` | no |
| <a name="input_service_account"></a> [service\_account](#input\_service\_account) | The email address of the service account to run the service as. | `string` | n/a | yes |
| <a name="input_slo_notification_channels"></a> [slo\_notification\_channels](#input\_slo\_notification\_channels) | A list of notification channels to send alerts to. | `list(string)` | `[]` | no |
| <a name="input_slo_policy_link"></a> [slo\_policy\_link](#input\_slo\_policy\_link) | An optional link to the SLO policy to include in the alert documentation. | `string` | `""` | no |
| <a name="input_slo_threshold"></a> [slo\_threshold](#input\_slo\_threshold) | The uptime percent required to meet the SLO for the service, expressed as a decimal in {0, 1} | `number` | `0.999` | no |
| <a name="input_timeout"></a> [timeout](#input\_timeout) | The timeout for the prober in seconds. | `string` | `"60s"` | no |
| <a name="input_working_dir"></a> [working\_dir](#input\_working\_dir) | The working directory that contains the importpath. | `string` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_alert_id"></a> [alert\_id](#output\_alert\_id) | n/a |
| <a name="output_slo_alert_id"></a> [slo\_alert\_id](#output\_slo\_alert\_id) | n/a |
| <a name="output_uptime_check"></a> [uptime\_check](#output\_uptime\_check) | n/a |
| <a name="output_uptime_check_name"></a> [uptime\_check\_name](#output\_uptime\_check\_name) | n/a |
<!-- END_TF_DOCS -->
