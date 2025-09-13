# Workqueue Dashboard Module

This module creates a Cloud Monitoring dashboard for a workqueue implementation, providing visibility into queue metrics, processing latency, and system health.

## Usage

```hcl
module "workqueue-dashboard" {
  source = "chainguard-dev/common/infra//modules/dashboard/workqueue"

  name            = var.name
  max_retry       = var.max-retry
  concurrent_work = var.concurrent-work
  scope           = var.scope

  labels = {
    team = "platform"
  }

  alerts = {
    "deadletter-alert" = google_monitoring_alert_policy.deadletter.id
  }
}
```

## Features

The dashboard includes:

- **Queue State**: Work in progress, queued items, and items added
- **Processing Metrics**: Processing latency, wait times, and deduplication rates
- **Retry Analytics**: Attempts at completion, maximum attempts, and time to completion
- **Dead Letter Queue**: Monitoring for items that exceeded retry limits (when `max_retry > 0`)
- **Service Logs**: Separate log views for receiver and dispatcher services
- **Alert Integration**: Display configured alerts on the dashboard

## Dashboard Filters

The dashboard automatically configures filters for:
- Receiver service (`${name}-rcv`)
- Dispatcher service (`${name}-dsp`)
- Both Cloud Run built-in metrics and Prometheus custom metrics
