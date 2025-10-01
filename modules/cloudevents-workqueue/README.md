# CloudEvents Workqueue Module

This Terraform module provisions a service that subscribes to CloudEvents from a broker and enqueues work items to a workqueue based on a specified CloudEvent extension attribute.

## Architecture

The module creates a **Subscriber Service** that:
- Subscribes to specific CloudEvent types from a broker (Pub/Sub topic)
- Extracts a workqueue key from a specified CloudEvent extension
- Enqueues work items to a workqueue service for processing

This module is designed to work with `github-events` module, which publishes GitHub webhook events as CloudEvents with extensions like `pullrequesturl` and `issueurl`.

## Key Features

- **Flexible Event Filtering**: Support multiple Knative Trigger-style filters with OR logic
- **Extension-based Routing**: Use any CloudEvent extension as the workqueue key
- **Reliable Delivery**: Built-in retry logic and error handling
- **Cloud-native**: Runs on Cloud Run with automatic scaling

## Setup Instructions

### Step 1: Deploy Prerequisites

You need:
1. A CloudEvent broker (from `cloudevent-broker` module)
2. A workqueue service (from `workqueue` module)
3. A source of CloudEvents (e.g., `github-events` module)

### Step 2: Deploy the CloudEvents Workqueue

```hcl
module "github_pr_processor" {
  source = "../../modules/cloudevents-workqueue"

  project_id            = var.project_id
  name                  = "github-pr-processor"
  regions               = var.regions
  notification_channels = var.notification_channels

  # Subscribe to the broker
  broker = module.cloudevent-broker.broker

  # Subscribe to specific CloudEvent types using filters
  filters = [
    { "type" = "dev.chainguard.github.pull_request" },
    { "type" = "dev.chainguard.github.pull_request_review" },
    { "type" = "dev.chainguard.github.pull_request_review_comment" },
    { "type" = "dev.chainguard.github.issue_comment" },  # For PR comments
  ]

  # Use the pullrequesturl extension as the workqueue key
  extension_key = "pullrequesturl"

  # Send to workqueue
  workqueue = {
    name = module.workqueue.dispatcher.name
  }
}
```

### Step 3: Process Work Items

Implement a workqueue consumer that processes the URLs:

```go
package main

import (
    "context"
    "log"

    "github.com/chainguard-dev/terraform-infra-common/pkg/workqueue"
)

func main() {
    ctx := context.Background()

    client, err := workqueue.NewWorkqueueClient(ctx, os.Getenv("WORKQUEUE_SERVICE"))
    if err != nil {
        log.Fatal(err)
    }

    for {
        item, err := client.Dequeue(ctx)
        if err != nil {
            log.Printf("Error dequeuing: %v", err)
            continue
        }

        // item.Key will be the PR URL (e.g., https://github.com/owner/repo/pull/123)
        if err := processPullRequest(ctx, item.Key); err != nil {
            log.Printf("Error processing %s: %v", item.Key, err)
        }
    }
}
```

## Use Cases

### GitHub Pull Request Processing

Subscribe to PR-related events and process them using the PR URL:

```hcl
module "pr_processor" {
  source = "../../modules/cloudevents-workqueue"

  # ... base configuration ...

  # List all PR-related events
  filters = [
    { "type" = "dev.chainguard.github.pull_request" },
    { "type" = "dev.chainguard.github.pull_request_review" },
    { "type" = "dev.chainguard.github.pull_request_review_comment" },
    { "type" = "dev.chainguard.github.check_run" },
    { "type" = "dev.chainguard.github.check_suite" }
  ]

  extension_key = "pullrequesturl"
}
```

### Advanced Filtering Examples

Filter only opened PRs:
```hcl
filters = [
  {
    "type"   = "dev.chainguard.github.pull_request"
    "action" = "opened"
  }
]
```

Filter only merged PRs:
```hcl
filters = [
  {
    "type"   = "dev.chainguard.github.pull_request"
    "action" = "closed"
    "merged" = "true"
  }
]
```

Filter multiple specific event types:
```hcl
filters = [
  { "type" = "dev.chainguard.github.pull_request" },
  { "type" = "dev.chainguard.github.issues" },
  { "type" = "dev.chainguard.github.check_run" },
  { "type" = "dev.chainguard.github.check_suite" }
]
```

### GitHub Issue Processing

Subscribe to issue-related events and process them using the issue URL:

```hcl
module "issue_processor" {
  source = "../../modules/cloudevents-workqueue"

  # ... base configuration ...

  # List all issue-related events
  filters = [
    { "type" = "dev.chainguard.github.issues" },
    { "type" = "dev.chainguard.github.issue_comment" }
  ]

  extension_key = "issueurl"
}
```

### Custom CloudEvents

This module works with any CloudEvents that have the appropriate extension:

```hcl
module "custom_processor" {
  source = "../../modules/cloudevents-workqueue"

  # ... base configuration ...

  # List specific user events
  filters = [
    { "type" = "com.example.user.created" },
    { "type" = "com.example.user.updated" },
    { "type" = "com.example.user.deleted" }
  ]

  # Use a custom extension
  extension_key = "userid"
}
```

## How It Works

1. **Event Reception**: The subscriber service receives CloudEvents via HTTP POST from Pub/Sub
2. **Extension Extraction**: The service looks for the specified extension in the CloudEvent
3. **Workqueue Enqueue**: If the extension exists and has a non-empty string value, it's enqueued
4. **Error Handling**:
   - Missing or invalid extensions are logged and acknowledged (no retry)
   - Workqueue errors trigger retries via Pub/Sub redelivery

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
| <a name="module_subscriber"></a> [subscriber](#module\_subscriber) | ../regional-go-service | n/a |
| <a name="module_subscriber-calls-workqueue"></a> [subscriber-calls-workqueue](#module\_subscriber-calls-workqueue) | ../authorize-private-service | n/a |
| <a name="module_trigger"></a> [trigger](#module\_trigger) | ../cloudevent-trigger | n/a |

## Resources

| Name | Type |
|------|------|
| [google_service_account.subscriber](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/service_account) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_ack_deadline_seconds"></a> [ack\_deadline\_seconds](#input\_ack\_deadline\_seconds) | The deadline for acking a message. | `number` | `300` | no |
| <a name="input_broker"></a> [broker](#input\_broker) | A map from each of the input region names to the name of the Broker topic in that region. | `map(string)` | n/a | yes |
| <a name="input_deletion_protection"></a> [deletion\_protection](#input\_deletion\_protection) | Whether to enable deletion protection for resources | `bool` | `true` | no |
| <a name="input_extension_key"></a> [extension\_key](#input\_extension\_key) | The CloudEvent extension attribute to use as the workqueue key (e.g., pullrequesturl or issueurl) | `string` | n/a | yes |
| <a name="input_filters"></a> [filters](#input\_filters) | A list of Knative Trigger-style filters over cloud event attributes.<br/><br/>Each filter is a map of attribute key-value pairs that must match exactly.<br/>Multiple filters are combined with OR logic (any filter can match).<br/><br/>Examples:<br/>  # Single event type<br/>  filters = [<br/>    { "type" = "dev.chainguard.github.pull\_request" }<br/>  ]<br/><br/>  # Multiple event types<br/>  filters = [<br/>    { "type" = "dev.chainguard.github.pull\_request" },<br/>    { "type" = "dev.chainguard.github.pull\_request\_review" }<br/>  ]<br/><br/>  # Filter by type and action<br/>  filters = [<br/>    {<br/>      "type"   = "dev.chainguard.github.pull\_request"<br/>      "action" = "opened"<br/>    }<br/>  ] | `list(map(string))` | `[]` | no |
| <a name="input_max_delivery_attempts"></a> [max\_delivery\_attempts](#input\_max\_delivery\_attempts) | The maximum number of delivery attempts for any event. | `number` | `20` | no |
| <a name="input_maximum_backoff"></a> [maximum\_backoff](#input\_maximum\_backoff) | The maximum delay between consecutive deliveries of a given message. | `number` | `600` | no |
| <a name="input_minimum_backoff"></a> [minimum\_backoff](#input\_minimum\_backoff) | The minimum delay between consecutive deliveries of a given message. | `number` | `10` | no |
| <a name="input_name"></a> [name](#input\_name) | The base name for resources | `string` | n/a | yes |
| <a name="input_notification_channels"></a> [notification\_channels](#input\_notification\_channels) | List of notification channels for alerts | `list(string)` | n/a | yes |
| <a name="input_product"></a> [product](#input\_product) | Product label to apply to the service. | `string` | `"unknown"` | no |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | The GCP project ID | `string` | n/a | yes |
| <a name="input_regions"></a> [regions](#input\_regions) | A map of regions to launch services in (see regional-go-service module for format) | <pre>map(object({<br/>    network = string<br/>    subnet  = string<br/>  }))</pre> | n/a | yes |
| <a name="input_squad"></a> [squad](#input\_squad) | squad label to apply to the service. | `string` | `"unknown"` | no |
| <a name="input_workqueue"></a> [workqueue](#input\_workqueue) | The workqueue to send events to | <pre>object({<br/>    name = string<br/>  })</pre> | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_subscriber"></a> [subscriber](#output\_subscriber) | n/a |
<!-- END_TF_DOCS -->
