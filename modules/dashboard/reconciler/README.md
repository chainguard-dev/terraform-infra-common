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
}
<!-- BEGIN_TF_DOCS -->
## Requirements

No requirements.

## Providers

No providers.

## Modules

| Name | Source | Version |
|------|--------|---------|
| <a name="module_agents"></a> [agents](#module\_agents) | ../sections/agents | n/a |
| <a name="module_alerts"></a> [alerts](#module\_alerts) | ../sections/alerts | n/a |
| <a name="module_dashboard"></a> [dashboard](#module\_dashboard) | ../ | n/a |
| <a name="module_errgrp"></a> [errgrp](#module\_errgrp) | ../sections/errgrp | n/a |
| <a name="module_github"></a> [github](#module\_github) | ../sections/github | n/a |
| <a name="module_grpc"></a> [grpc](#module\_grpc) | ../sections/grpc | n/a |
| <a name="module_http"></a> [http](#module\_http) | ../sections/http | n/a |
| <a name="module_layout"></a> [layout](#module\_layout) | ../sections/layout | n/a |
| <a name="module_reconciler-logs"></a> [reconciler-logs](#module\_reconciler-logs) | ../sections/logs | n/a |
| <a name="module_resources"></a> [resources](#module\_resources) | ../sections/resources | n/a |
| <a name="module_width"></a> [width](#module\_width) | ../sections/width | n/a |
| <a name="module_workqueue-state"></a> [workqueue-state](#module\_workqueue-state) | ../sections/workqueue | n/a |

## Resources

No resources.

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_alerts"></a> [alerts](#input\_alerts) | Map of alert names to alert configurations | <pre>map(object({<br/>    displayName         = string<br/>    documentation       = string<br/>    userLabels          = map(string)<br/>    project             = string<br/>    notificationChannel = string<br/>  }))</pre> | `{}` | no |
| <a name="input_concurrent_work"></a> [concurrent\_work](#input\_concurrent\_work) | The amount of concurrent work the workqueue dispatches | `number` | `20` | no |
| <a name="input_labels"></a> [labels](#input\_labels) | Additional labels to add to the dashboard | `map(string)` | `{}` | no |
| <a name="input_max_retry"></a> [max\_retry](#input\_max\_retry) | The maximum number of retry attempts for workqueue tasks | `number` | `100` | no |
| <a name="input_name"></a> [name](#input\_name) | The name of the reconciler (base name without suffixes) | `string` | n/a | yes |
| <a name="input_notification_channels"></a> [notification\_channels](#input\_notification\_channels) | List of notification channels for alerts | `list(string)` | `[]` | no |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | The GCP project ID | `string` | n/a | yes |
| <a name="input_sections"></a> [sections](#input\_sections) | Configure visibility of optional dashboard sections | <pre>object({<br/>    github = optional(bool, false)<br/>    agents = optional(bool, false)<br/>  })</pre> | `{}` | no |
| <a name="input_service_name"></a> [service\_name](#input\_service\_name) | The name of the reconciler service (defaults to name-rec) | `string` | `""` | no |
| <a name="input_shards"></a> [shards](#input\_shards) | Number of workqueue shards. When > 1, dashboard shows per-shard metrics. | `number` | `1` | no |
| <a name="input_workqueue_name"></a> [workqueue\_name](#input\_workqueue\_name) | The name of the workqueue (defaults to name-wq) | `string` | `""` | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_json"></a> [json](#output\_json) | n/a |
<!-- END_TF_DOCS -->
