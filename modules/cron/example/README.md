# Cron Module Example

This example demonstrates how to use the cron module to deploy a scheduled Cloud Run job with success alerting.

## Features Demonstrated

1. **Basic cron job setup** - Deploys a simple Go application that runs every 8 minutes
2. **Success alerting** - Configures monitoring to alert when the job fails to complete successfully
3. **Separate duration and alignment** - Shows how to use different values for detection speed vs monitoring window

## Alert Configuration

The example configures success alerting with:
- **Alignment period**: 30 minutes (looks for any successful execution in the past 30 minutes)
- **Duration**: 15 minutes (alerts after 15 minutes of no successful executions)

This configuration provides:
- Fast detection of failures (15 minutes)
- While checking a broader context window (30 minutes)
- Suitable for jobs that run every 8 minutes

## Usage

```bash
terraform init
terraform apply -var="project_id=your-project-id"
```

## Note

Remember that the alert will not fire until the job has completed successfully at least once. This is a GCP requirement for metric-absence conditions.

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
| <a name="module_cron"></a> [cron](#module\_cron) | ../ | n/a |

## Resources

| Name | Type |
|------|------|
| [google_service_account.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/service_account) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | The project that will host the cron job. | `string` | n/a | yes |

## Outputs

No outputs.
<!-- END_TF_DOCS -->
