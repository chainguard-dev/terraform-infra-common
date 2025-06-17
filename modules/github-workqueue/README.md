# GitHub Workqueue Module

This Terraform module provisions a GitHub webhook service that receives GitHub events and enqueues them to a workqueue for processing. It acts as a bridge between GitHub webhooks and the workqueue service, allowing you to process GitHub events asynchronously.

## Architecture

The module creates a **Webhook Service** that:
- Receives GitHub webhook events via HTTP (through a load balancer)
- Validates webhook signatures for security
- Parses events to extract resource URLs (issues/PRs)
- Enqueues work items to a workqueue service

The webhook service is configured to only accept traffic from Google Cloud Load Balancers (GCLB). You'll need to configure a load balancer separately to expose the service publicly.

The actual processing of these events is left to the user to implement as a separate workqueue consumer service.

## Usage

### Basic Module Usage

```hcl
module "github_webhook" {
  source = "../../modules/github-workqueue"

  project_id            = var.project_id
  name                  = "my-github-webhook"
  regions               = var.regions
  notification_channels = var.notification_channels

  # Workqueue configuration
  workqueue = {
    name = "${var.name}-dsp"  # The dispatcher service name from the workqueue module
  }

  # Optional: filter to only process specific resource types
  resource_filter = "pull_requests" # or "issues" or "" (no filter)
}
```

### Supported GitHub Events

The webhook service extracts resource URLs from the following GitHub events:

- **Issues**: `issues` events
- **Issue Comments**: `issue_comment` events (for both issues and PRs)
- **Pull Requests**: `pull_request` events
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
| <a name="module_shim"></a> [shim](#module\_shim) | ../regional-go-service | n/a |
| <a name="module_shim-calls-workqueue"></a> [shim-calls-workqueue](#module\_shim-calls-workqueue) | ../authorize-private-service | n/a |
| <a name="module_webhook-secret"></a> [webhook-secret](#module\_webhook-secret) | ../configmap | n/a |

## Resources

| Name | Type |
|------|------|
| [google_service_account.shim](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/service_account) | resource |
| [random_password.webhook-secret](https://registry.terraform.io/providers/hashicorp/random/latest/docs/resources/password) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_name"></a> [name](#input\_name) | The base name for resources | `string` | n/a | yes |
| <a name="input_notification_channels"></a> [notification\_channels](#input\_notification\_channels) | List of notification channels for alerts | `list(string)` | n/a | yes |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | The GCP project ID | `string` | n/a | yes |
| <a name="input_regions"></a> [regions](#input\_regions) | A map of regions to launch services in (see regional-go-service module for format) | <pre>map(object({<br/>    network = string<br/>    subnet  = string<br/>  }))</pre> | n/a | yes |
| <a name="input_resource_filter"></a> [resource\_filter](#input\_resource\_filter) | Optional filter to process only specific resource types | `string` | `""` | no |
| <a name="input_workqueue"></a> [workqueue](#input\_workqueue) | The workqueue to send events to | <pre>object({<br/>    name = string<br/>  })</pre> | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_webhook"></a> [webhook](#output\_webhook) | n/a |
<!-- END_TF_DOCS -->
