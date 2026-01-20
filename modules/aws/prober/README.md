# AWS Prober Module

This Terraform module deploys a prober service to AWS App Runner with integrated uptime monitoring using CloudWatch Synthetics. It's the AWS equivalent of the GCP `prober` module.

## Features

- **Automated Prober Deployment**: Deploys Go-based prober applications to AWS App Runner
- **Shared Secret Authentication**: Generates a random password for authorization headers to prevent abuse
- **CloudWatch Synthetics Integration**: Optional canary monitoring for uptime checks
- **CloudWatch Alarms**: Configurable alerting on uptime check failures
- **Automatic IAM Setup**: Creates all necessary IAM roles and policies automatically
- **VPC Support**: Optional VPC connector for accessing private resources
- **X-Ray Tracing**: Optional AWS X-Ray integration for observability

## Usage

### Basic Example

```hcl
module "my_prober" {
  source = "./modules/aws/prober"

  name    = "api-health"
  team    = "platform"
  product = "monitoring"

  importpath  = "github.com/my-org/my-prober"
  working_dir = "${path.module}/prober"

  # Environment variables for the prober
  env = {
    TARGET_URL = "https://api.example.com"
    CHECK_TYPE = "http"
  }

  # Enable CloudWatch Synthetics for uptime monitoring
  cloudwatch_synthetics_enabled = true
  canary_schedule              = "rate(5 minutes)"

  # Enable alerting
  enable_alert          = true
  notification_channels = [aws_sns_topic.alerts.arn]
}
```

### With VPC Access

For probers that need to access private resources:

```hcl
# Create VPC connector
resource "aws_apprunner_vpc_connector" "prober" {
  vpc_connector_name = "prober-vpc-connector"
  subnets            = var.private_subnet_ids
  security_groups    = [aws_security_group.prober.id]
}

module "internal_prober" {
  source = "./modules/aws/prober"

  name    = "internal-api-health"
  team    = "platform"
  product = "monitoring"

  importpath  = "github.com/my-org/my-prober"
  working_dir = "${path.module}/prober"

  # VPC configuration for private resource access
  egress            = "VPC"
  vpc_connector_arn = aws_apprunner_vpc_connector.prober.arn

  env = {
    TARGET_URL = "https://internal-api.private.example.com"
  }
}
```

### With Secrets

For probers that need access to secrets:

```hcl
# Create secrets
resource "aws_secretsmanager_secret" "api_key" {
  name = "prober-api-key"
}

resource "aws_secretsmanager_secret_version" "api_key" {
  secret_id     = aws_secretsmanager_secret.api_key.id
  secret_string = "your-secret-key"
}

module "authenticated_prober" {
  source = "./modules/aws/prober"

  name    = "authenticated-api-health"
  team    = "platform"
  product = "monitoring"

  importpath  = "github.com/my-org/my-prober"
  working_dir = "${path.module}/prober"

  env = {
    TARGET_URL = "https://api.example.com"
  }

  # Mount secrets as environment variables
  secret_env = {
    API_KEY = aws_secretsmanager_secret.api_key.arn
  }
}
```

### Custom Resource Sizing

```hcl
module "heavy_prober" {
  source = "./modules/aws/prober"

  name    = "load-test-prober"
  team    = "platform"
  product = "monitoring"

  importpath  = "github.com/my-org/my-prober"
  working_dir = "${path.module}/prober"

  # Increase resources for heavy workloads
  cpu    = 2048  # 2 vCPU
  memory = 4096  # 4 GB

  # Adjust scaling
  scaling = {
    min_instances                    = 2
    max_instances                    = 10
    max_instance_request_concurrency = 50
  }
}
```

## How It Works

1. **Prober Service**: Deploys your Go prober application to AWS App Runner
2. **Shared Secret**: Generates a random authorization token that's passed to both:
   - The prober service as an environment variable (`AUTHORIZATION`)
   - The CloudWatch Synthetics canary as a custom header
3. **Uptime Check**: CloudWatch Synthetics canary periodically hits the prober endpoint with the authorization header
4. **Alerting**: CloudWatch Alarms monitor the canary success rate and send notifications on failures

## Prober Application Requirements

Your prober application should:

1. Listen on port 8080
2. Respond to GET requests on `/`
3. Verify the `Authorization` header matches the shared secret
4. Return HTTP 200 on success, non-200 on failure

Example Go prober:

```go
package main

import (
    "fmt"
    "log"
    "net/http"
    "os"
)

func main() {
    expectedAuth := os.Getenv("AUTHORIZATION")

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        // Verify authorization
        if r.Header.Get("Authorization") != expectedAuth {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }

        // Perform your health checks here
        if err := checkTargetHealth(); err != nil {
            http.Error(w, err.Error(), http.StatusServiceUnavailable)
            return
        }

        fmt.Fprintf(w, "OK")
    })

    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

## CloudWatch Synthetics

The module uses CloudWatch Synthetics to create a canary that:
- Runs on a configurable schedule (default: every 5 minutes)
- Makes HTTPS requests to the prober endpoint
- Includes the authorization header
- Reports success/failure metrics to CloudWatch

## Monitoring and Alerting

### Metrics

The canary publishes metrics to CloudWatch under the `CloudWatchSynthetics` namespace:
- `SuccessPercent`: Percentage of successful checks
- `Duration`: Time taken for each check
- `Failed`: Number of failed checks

### Alarms

When `enable_alert = true`, the module creates a CloudWatch alarm that:
- Monitors the `SuccessPercent` metric
- Triggers when success rate drops below 90% for 2 evaluation periods
- Sends notifications to configured SNS topics


<!-- BEGIN_TF_DOCS -->
## Requirements

No requirements.

## Providers

| Name | Version |
|------|---------|
| <a name="provider_archive"></a> [archive](#provider\_archive) | n/a |
| <a name="provider_aws"></a> [aws](#provider\_aws) | n/a |
| <a name="provider_random"></a> [random](#provider\_random) | n/a |

## Modules

| Name | Source | Version |
|------|--------|---------|
| <a name="module_this"></a> [this](#module\_this) | ../apprunner-regional-go-service | n/a |

## Resources

| Name | Type |
|------|------|
| [aws_cloudwatch_metric_alarm.uptime_alert](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/cloudwatch_metric_alarm) | resource |
| [aws_iam_role.canary](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role) | resource |
| [aws_iam_role_policy.canary_permissions](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role_policy) | resource |
| [aws_iam_role_policy_attachment.canary_xray](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role_policy_attachment) | resource |
| [aws_s3_bucket.canary_artifacts](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/s3_bucket) | resource |
| [aws_synthetics_canary.uptime_check](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/synthetics_canary) | resource |
| [random_password.secret](https://registry.terraform.io/providers/hashicorp/random/latest/docs/resources/password) | resource |
| [archive_file.canary_script](https://registry.terraform.io/providers/hashicorp/archive/latest/docs/data-sources/file) | data source |
| [aws_caller_identity.current](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/caller_identity) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_alarm_comparison_operator"></a> [alarm\_comparison\_operator](#input\_alarm\_comparison\_operator) | The arithmetic operation to use when comparing the specified statistic and threshold. Valid values: GreaterThanOrEqualToThreshold, GreaterThanThreshold, LessThanThreshold, LessThanOrEqualToThreshold. | `string` | `"LessThanThreshold"` | no |
| <a name="input_alarm_datapoints_to_alarm"></a> [alarm\_datapoints\_to\_alarm](#input\_alarm\_datapoints\_to\_alarm) | The number of datapoints that must be breaching to trigger the alarm. Defaults to evaluation\_periods if not set. | `number` | `null` | no |
| <a name="input_alarm_evaluation_periods"></a> [alarm\_evaluation\_periods](#input\_alarm\_evaluation\_periods) | The number of periods over which data is compared to the specified threshold. | `number` | `2` | no |
| <a name="input_alarm_statistic"></a> [alarm\_statistic](#input\_alarm\_statistic) | The statistic to apply to the alarm's associated metric. Valid values: SampleCount, Average, Sum, Minimum, Maximum. | `string` | `"Average"` | no |
| <a name="input_alarm_threshold"></a> [alarm\_threshold](#input\_alarm\_threshold) | The value against which the specified statistic is compared. For SuccessPercent, this is the percentage (0-100). | `number` | `90` | no |
| <a name="input_alarm_treat_missing_data"></a> [alarm\_treat\_missing\_data](#input\_alarm\_treat\_missing\_data) | How to handle missing data points. Valid values: missing, ignore, breaching, notBreaching. | `string` | `"notBreaching"` | no |
| <a name="input_alert_description"></a> [alert\_description](#input\_alert\_description) | Alert documentation. Use this to link to playbooks or give additional context. | `string` | `"An uptime check has failed."` | no |
| <a name="input_base_image"></a> [base\_image](#input\_base\_image) | The base image to use for the prober. | `string` | `null` | no |
| <a name="input_canary_runtime_version"></a> [canary\_runtime\_version](#input\_canary\_runtime\_version) | CloudWatch Synthetics runtime version. | `string` | `"syn-nodejs-puppeteer-13.0"` | no |
| <a name="input_canary_schedule"></a> [canary\_schedule](#input\_canary\_schedule) | CloudWatch Synthetics canary schedule expression. | `string` | `"rate(5 minutes)"` | no |
| <a name="input_cloudwatch_synthetics_enabled"></a> [cloudwatch\_synthetics\_enabled](#input\_cloudwatch\_synthetics\_enabled) | Enable CloudWatch Synthetics canary for uptime monitoring. | `bool` | `true` | no |
| <a name="input_cpu"></a> [cpu](#input\_cpu) | The CPU units for the prober. Valid values: 256, 512, 1024, 2048, 4096 | `number` | `1024` | no |
| <a name="input_egress"></a> [egress](#input\_egress) | Network egress configuration. DEFAULT for internet, VPC for private resources | `string` | `"DEFAULT"` | no |
| <a name="input_enable_alert"></a> [enable\_alert](#input\_enable\_alert) | If true, alert on failures. Outputs will return the alert ID for notification and dashboards. | `bool` | `false` | no |
| <a name="input_enable_profiler"></a> [enable\_profiler](#input\_enable\_profiler) | Enable cloud profiler (AWS X-Ray). | `bool` | `false` | no |
| <a name="input_env"></a> [env](#input\_env) | A map of custom environment variables (e.g. key=value) | `map(string)` | `{}` | no |
| <a name="input_importpath"></a> [importpath](#input\_importpath) | The import path that contains the prober application. | `string` | n/a | yes |
| <a name="input_ingress"></a> [ingress](#input\_ingress) | Network ingress configuration. PUBLIC for internet access, PRIVATE for VPC only | `string` | `"PUBLIC"` | no |
| <a name="input_memory"></a> [memory](#input\_memory) | The memory in MB for the prober. Valid values: 512, 1024, 2048, 3072, 4096, 6144, 8192, 10240, 12288 | `number` | `2048` | no |
| <a name="input_name"></a> [name](#input\_name) | Name to prefix to created resources. | `string` | n/a | yes |
| <a name="input_notification_channels"></a> [notification\_channels](#input\_notification\_channels) | A list of SNS topic ARNs to send alerts to. | `list(string)` | `[]` | no |
| <a name="input_period"></a> [period](#input\_period) | The period for the prober in seconds. Supported values: 60s (1 minute), 300s (5 minutes), 600s (10 minutes), and 900s (15 minutes) | `string` | `"300s"` | no |
| <a name="input_product"></a> [product](#input\_product) | Product label to apply to the service. | `string` | n/a | yes |
| <a name="input_scaling"></a> [scaling](#input\_scaling) | The scaling configuration for the service. | <pre>object({<br/>    min_instances                    = optional(number, 1)<br/>    max_instances                    = optional(number, 25)<br/>    max_instance_request_concurrency = optional(number, 100)<br/>  })</pre> | `{}` | no |
| <a name="input_secret_env"></a> [secret\_env](#input\_secret\_env) | A map of secrets to mount as environment variables from AWS Secrets Manager or SSM Parameter Store (e.g. secret\_key=secret\_arn) | `map(string)` | `{}` | no |
| <a name="input_start_canary"></a> [start\_canary](#input\_start\_canary) | Automatically start the canary after creation. Set to false to create the canary in a stopped state. | `bool` | `true` | no |
| <a name="input_tags"></a> [tags](#input\_tags) | Additional tags to apply to resources | `map(string)` | `{}` | no |
| <a name="input_team"></a> [team](#input\_team) | Team label to apply to resources. | `string` | n/a | yes |
| <a name="input_timeout"></a> [timeout](#input\_timeout) | The timeout for the prober in seconds. Supported values 1-60s | `string` | `"60s"` | no |
| <a name="input_uptime_alert_duration"></a> [uptime\_alert\_duration](#input\_uptime\_alert\_duration) | Duration for uptime alert policy. | `string` | `"600s"` | no |
| <a name="input_vpc_connector_arn"></a> [vpc\_connector\_arn](#input\_vpc\_connector\_arn) | Optional VPC connector ARN for private resource access (required if egress is VPC). | `string` | `null` | no |
| <a name="input_working_dir"></a> [working\_dir](#input\_working\_dir) | The working directory that contains the importpath. | `string` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_alarm_arn"></a> [alarm\_arn](#output\_alarm\_arn) | CloudWatch alarm ARN (if enabled) |
| <a name="output_authorization_secret"></a> [authorization\_secret](#output\_authorization\_secret) | The shared secret used for authorization (sensitive) |
| <a name="output_canary_arn"></a> [canary\_arn](#output\_canary\_arn) | CloudWatch Synthetics canary ARN (if enabled) |
| <a name="output_canary_name"></a> [canary\_name](#output\_canary\_name) | CloudWatch Synthetics canary name (if enabled) |
| <a name="output_service_arn"></a> [service\_arn](#output\_service\_arn) | App Runner service ARN |
| <a name="output_service_name"></a> [service\_name](#output\_service\_name) | App Runner service name |
| <a name="output_service_url"></a> [service\_url](#output\_service\_url) | App Runner service URL |
<!-- END_TF_DOCS -->
