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

## Setup Instructions

### Step 1: Deploy the Workqueue Infrastructure

First, you need a workqueue service to send events to. Use the workqueue module:

```hcl
module "workqueue" {
  source = "../../modules/workqueue"

  project_id            = var.project_id
  name                  = "${var.name}-queue"
  regions               = var.regions
  notification_channels = var.notification_channels
}
```

### Step 2: Deploy the GitHub Webhook Service

```hcl
module "github_webhook" {
  source = "../../modules/github-workqueue"

  project_id            = var.project_id
  name                  = "${var.name}-webhook"
  regions               = var.regions
  notification_channels = var.notification_channels

  # Reference the workqueue dispatcher
  workqueue = {
    name = module.workqueue.dispatcher.name
  }

  # Optional: filter to only process specific resource types
  resource_filter = "pull_requests" # or "issues" or "" (no filter)
}
```

### Step 3: Configure Load Balancer (Required)

The webhook service only accepts traffic from Google Cloud Load Balancers. You need to create a load balancer to expose the service:

```hcl
# Example load balancer configuration
resource "google_compute_global_address" "webhook" {
  name         = "${var.name}-webhook-ip"
  project      = var.project_id
  address_type = "EXTERNAL"
}

# Configure your load balancer to route traffic to the webhook service
# The service is available at module.github_webhook.webhook.uri
```

### Step 4: Configure GitHub Webhook

1. **Get the webhook secret**:
   ```bash
   gcloud secrets versions access latest --secret="${module.github_webhook.webhook.secret_id}" --project="${var.project_id}"
   ```

2. **Configure the webhook in GitHub**:
   - Go to your GitHub repository or organization settings
   - Navigate to **Settings** → **Webhooks** → **Add webhook**
   - **Payload URL**: `https://your-domain.com/webhook` (your load balancer URL)
   - **Content type**: `application/json`
   - **Secret**: Use the secret retrieved above
   - **Events**: Select the events you want to receive:
     - Issues
     - Issue comments
     - Pull requests
     - Pull request reviews
     - Pull request review comments
     - Check runs
     - Check suites

### Step 5: Implement a Workqueue Consumer

Create a service that processes the queued GitHub events:

```go
// Example consumer implementation
package main

import (
    "context"
    "log"

    "github.com/chainguard-dev/terraform-infra-common/pkg/githubreconciler"
    "github.com/chainguard-dev/terraform-infra-common/pkg/workqueue"
)

func main() {
    ctx := context.Background()

    // Connect to workqueue
    client, err := workqueue.NewWorkqueueClient(ctx, os.Getenv("WORKQUEUE_SERVICE"))
    if err != nil {
        log.Fatal(err)
    }

    // Process items
    for {
        item, err := client.Dequeue(ctx)
        if err != nil {
            log.Printf("Error dequeuing: %v", err)
            continue
        }

        // Parse the GitHub URL
        resource, err := githubreconciler.ParseURL(item.Key)
        if err != nil {
            log.Printf("Error parsing URL: %v", err)
            continue
        }

        // Process the resource (implement your logic here)
        if err := processResource(ctx, resource); err != nil {
            log.Printf("Error processing %s: %v", item.Key, err)
            // Optionally requeue for retry
        }
    }
}
```

See the `examples/` directory for complete reconciler implementations.

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

### Environment Variables

The webhook service uses these environment variables (automatically configured by Terraform):
- `WORKQUEUE_SERVICE`: The workqueue dispatcher service URL
- `GITHUB_WEBHOOK_SECRET`: The webhook validation secret
- `RESOURCE_FILTER`: Optional filter for resource types
- `PORT`: The port to listen on (default: 8080)

### Supported GitHub Events

The webhook service extracts resource URLs from the following GitHub events:

- **Issues**: `issues` events
- **Issue Comments**: `issue_comment` events (for both issues and PRs)
- **Pull Requests**: `pull_request` events
- **Pull Request Reviews**: `pull_request_review` events
- **Pull Request Review Comments**: `pull_request_review_comment` events
- **Check Runs**: `check_run` events (when associated with PRs)
- **Check Suites**: `check_suite` events (when associated with PRs)

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
| <a name="input_require_squad"></a> [require\_squad](#input\_require\_squad) | Whether to require squad variable to be specified | `bool` | `false` | no |
| <a name="input_resource_filter"></a> [resource\_filter](#input\_resource\_filter) | Optional filter to process only specific resource types | `string` | `""` | no |
| <a name="input_squad"></a> [squad](#input\_squad) | squad label to apply to the service. | `string` | `""` | no |
| <a name="input_workqueue"></a> [workqueue](#input\_workqueue) | The workqueue to send events to | <pre>object({<br/>    name = string<br/>  })</pre> | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_webhook"></a> [webhook](#output\_webhook) | n/a |
<!-- END_TF_DOCS -->
