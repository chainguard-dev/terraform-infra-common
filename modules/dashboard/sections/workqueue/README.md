# Workqueue Section Module

This module creates a dashboard section for monitoring workqueue state, including queue metrics, processing latency, and retry patterns.

## Usage

```hcl
module "workqueue-state" {
  source = "../sections/workqueue"

  title           = "My Workqueue"
  service_name    = "my-service"
  max_retry       = 5
  concurrent_work = 10
  scope           = "regional"
  filter          = []
  collapsed       = false
}
```

## Variables

- `title` - Section title (default: "Workqueue State")
- `service_name` - Base name of the workqueue service
- `receiver_name` - Optional explicit receiver service name (defaults to `${service_name}-rcv`)
- `dispatcher_name` - Optional explicit dispatcher service name (defaults to `${service_name}-dsp`)
- `max_retry` - Maximum retry limit for display thresholds
- `concurrent_work` - Concurrent work limit for display thresholds
- `scope` - Workqueue scope: "regional" or "global"
- `filter` - Additional metric filters to apply
- `collapsed` - Whether the section starts collapsed

## Included Metrics

The section displays:
- Work in progress, queued, and added
- Processing and wait latency
- Deduplication percentage
- Attempts at completion (95th percentile)
- Maximum task attempts
- Time to completion by priority
- Dead letter queue size (when max_retry > 0)

<!-- BEGIN_TF_DOCS -->
## Requirements

No requirements.

## Providers

No providers.

## Modules

| Name | Source | Version |
|------|--------|---------|
| <a name="module_attempts-at-completion"></a> [attempts-at-completion](#module\_attempts-at-completion) | ../../widgets/xy | n/a |
| <a name="module_collapsible"></a> [collapsible](#module\_collapsible) | ../collapsible | n/a |
| <a name="module_dead-letter-queue"></a> [dead-letter-queue](#module\_dead-letter-queue) | ../../widgets/xy | n/a |
| <a name="module_enumerate-latency"></a> [enumerate-latency](#module\_enumerate-latency) | ../../widgets/latency | n/a |
| <a name="module_expired-leases"></a> [expired-leases](#module\_expired-leases) | ../../widgets/xy | n/a |
| <a name="module_lease-age"></a> [lease-age](#module\_lease-age) | ../../widgets/latency | n/a |
| <a name="module_max-attempts"></a> [max-attempts](#module\_max-attempts) | ../../widgets/xy | n/a |
| <a name="module_percent-deduped"></a> [percent-deduped](#module\_percent-deduped) | ../../widgets/xy-ratio | n/a |
| <a name="module_process-latency"></a> [process-latency](#module\_process-latency) | ../../widgets/latency | n/a |
| <a name="module_time-to-completion"></a> [time-to-completion](#module\_time-to-completion) | ../../widgets/xy | n/a |
| <a name="module_time-until-eligible"></a> [time-until-eligible](#module\_time-until-eligible) | ../../widgets/latency | n/a |
| <a name="module_wait-latency"></a> [wait-latency](#module\_wait-latency) | ../../widgets/latency | n/a |
| <a name="module_width"></a> [width](#module\_width) | ../width | n/a |
| <a name="module_work-added"></a> [work-added](#module\_work-added) | ../../widgets/xy | n/a |
| <a name="module_work-in-progress"></a> [work-in-progress](#module\_work-in-progress) | ../../widgets/xy | n/a |
| <a name="module_work-queued"></a> [work-queued](#module\_work-queued) | ../../widgets/xy | n/a |

## Resources

No resources.

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_collapsed"></a> [collapsed](#input\_collapsed) | n/a | `bool` | `false` | no |
| <a name="input_concurrent_work"></a> [concurrent\_work](#input\_concurrent\_work) | n/a | `number` | n/a | yes |
| <a name="input_dispatcher_name"></a> [dispatcher\_name](#input\_dispatcher\_name) | n/a | `string` | `""` | no |
| <a name="input_filter"></a> [filter](#input\_filter) | n/a | `list(string)` | `[]` | no |
| <a name="input_max_retry"></a> [max\_retry](#input\_max\_retry) | n/a | `number` | `0` | no |
| <a name="input_receiver_name"></a> [receiver\_name](#input\_receiver\_name) | n/a | `string` | `""` | no |
| <a name="input_service_name"></a> [service\_name](#input\_service\_name) | n/a | `string` | n/a | yes |
| <a name="input_title"></a> [title](#input\_title) | n/a | `string` | `"Workqueue State"` | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_section"></a> [section](#output\_section) | n/a |
<!-- END_TF_DOCS -->
