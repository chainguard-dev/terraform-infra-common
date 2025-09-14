# Reconciler Dashboard Module

This module creates a comprehensive dashboard for monitoring a reconciler that combines a workqueue with a reconciler service. It displays metrics from both the workqueue infrastructure (receiver and dispatcher) and the reconciler service itself.

## Usage

```hcl
module "my-reconciler-dashboard" {
  source = "chainguard-dev/terraform-infra-common//modules/dashboard/reconciler"

  project_id = var.project_id
  name       = "my-reconciler"

  # Optional: Override service names if different from defaults
  # service_name   = "custom-reconciler-name"  # defaults to ${name}-rec
  # workqueue_name = "custom-workqueue-name"    # defaults to ${name}-wq

  # Workqueue configuration
  max_retry       = 100
  concurrent_work = 20
  scope           = "global"

  # Optional sections
  sections = {
    github = false
  }

  notification_channels = [var.notification_channel]
}
```

## Features

The dashboard includes:

### Workqueue Metrics
- **Workqueue State**: Work in progress, queued, added, deduplication rates, completion attempts
- **Processing Metrics**: Process latency, wait latency, time to completion
- **Dead Letter Queue**: Failed tasks monitoring

### Reconciler Service Metrics
- **Error Reporting**: Error tracking and reporting for the reconciler (collapsed by default)
- **Service Logs**: Reconciler service logs
- **gRPC Metrics**: RPC rates, latencies, error rates
- **GitHub API Metrics**: API usage and rate limiting (optional)
- **Resources**: CPU, memory, and other resource utilization

## Variables

| Name | Description | Default |
|------|-------------|---------|
| `project_id` | The GCP project ID | Required |
| `name` | Base name for the reconciler | Required |
| `service_name` | Reconciler service name | `${name}-rec` |
| `workqueue_name` | Workqueue name | `${name}-wq` |
| `max_retry` | Maximum retry attempts for tasks | `100` |
| `concurrent_work` | Concurrent work items | `20` |
| `scope` | Workqueue scope (regional/global) | `global` |
| `sections` | Optional dashboard sections | See variables.tf |
| `notification_channels` | Alert notification channels | `[]` |

## Outputs

| Name | Description |
|------|-------------|
| `json` | The dashboard JSON configuration |

## Integration with regional-go-reconciler

This dashboard module is designed to work seamlessly with the `regional-go-reconciler` module:

```hcl
module "my-reconciler" {
  source = "chainguard-dev/terraform-infra-common//modules/regional-go-reconciler"
  # ... configuration ...
}

module "my-reconciler-dashboard" {
  source = "chainguard-dev/terraform-infra-common//modules/dashboard/reconciler"

  project_id      = var.project_id
  name            = "my-reconciler"  # Same base name as the reconciler
  max_retry       = module.my-reconciler.max-retry
  concurrent_work = module.my-reconciler.concurrent-work
  scope           = "global"
}