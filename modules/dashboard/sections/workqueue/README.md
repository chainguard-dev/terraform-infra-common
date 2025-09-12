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
