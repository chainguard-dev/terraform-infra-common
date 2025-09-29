# `workqueue`

This module provisions a regionalized workqueue abstraction over Google Cloud
Storage that implements a Kubernetes-like workqueue abstraction for processing
work with concurrency control, in an otherwise stateless fashion (using Google
Cloud Run).

Keys are put into the queue and processed from the queue using a symmetrical
GRPC service definition under `./pkg/workqueue/workqueue.proto`.  This module
takes the name of a service implementing this proto service, and exposes the
name of a service into which work can be enqueued.

```hcl
module "workqueue" {
  source = "chainguard-dev/common/infra//modules/workqueue"

  project_id = var.project_id
  name       = "${var.name}-workqueue"
  regions    = var.regions

  // The number of keys to process concurrently.
  concurrent-work = 10

  // Maximum number of retry attempts before a task is moved to the dead letter queue
  // Default is 0 (unlimited retries)
  max-retry = 5

  // It is recommended that folks use a "global" scoped workqueue to get the
  // most accurate deduplication and concurrency control.  The "regional" scope
  // offers regionalized deduplication and concurrency control, but cannot
  // guarantee that receivers and dispatchers in other regions will not process
  // the same key concurrently or redundantly.
  scope = "global"

  // The name of a service that implements the workqueue GRPC service above.
  reconciler-service = {
    name = "foo"
  }

  notification_channels = var.notification_channels
}

// Authorize the bar service to queue keys in our workqueue.
module "bar-queues-keys" {
  for_each = var.regions

  source = "chainguard-dev/common/infra//modules/authorize-private-service"

  project_id = var.project_id
  region     = each.key
  name       = module.workqueue.receiver.name

  service-account = google_service_account.fanout.email
}

// Stand up the bar service in each of our regions.
module "bar-service" {
  source = "chainguard-dev/common/infra//modules/regional-go-service"
  ...
      regional-env = [{
        name  = "WORKQUEUE_SERVICE"
        value = { for k, v in module.bar-queues-keys : k => v.uri }
      }]
  ...
}
```

Then the "bar" service initializes a client for the workqueue GRPC service
pointing at `WORKQUEUE_SERVICE` with Cloud Run authentication (see the
`workqueue.NewWorkqueueClient` helper), and queues keys, e.g.

```go
	// Set up the client
	client, err := workqueue.NewWorkqueueClient(ctx, os.Getenv("WORKQUEUE_SERVICE"))
	if err != nil {
		log.Panicf("failed to create client: %v", err)
	}
	defer client.Close()

	// Process a key!
	if _, err := client.Process(ctx, &workqueue.ProcessRequest{
		Key: key,
	}); err != nil {
		log.Panicf("failed to process key: %v", err)
	}
```

## Dashboard

A separate dashboard module is available for monitoring your workqueue. The dashboard provides comprehensive visibility into queue metrics, processing latency, retry patterns, and system health.

```hcl
// Deploy the workqueue dashboard separately
module "workqueue-dashboard" {
  source = "chainguard-dev/common/infra//modules/dashboard/workqueue"

  // Pass the same configuration used for the workqueue
  name            = var.name
  max_retry       = var.max-retry
  concurrent_work = var.concurrent-work
  scope           = var.scope

  // Optional: Add alert policy IDs
  alerts = {
    "high-retry-alert" = google_monitoring_alert_policy.high_retry.id
  }
}
```

The dashboard includes:
- Queue state visualization (work in progress, queued, added)
- Processing and wait latency metrics
- Retry analytics and completion patterns
- Dead letter queue monitoring (when max-retry is configured)
- Service logs for receiver and dispatcher

See [`modules/dashboard/workqueue`](../dashboard/workqueue/) for more details.

## Maximum Retry and Dead Letter Queue

The workqueue system supports a maximum retry limit for tasks through the `max-retry` variable. When a task fails and gets requeued, the system tracks the number of attempts. Once the maximum retry limit is reached, the task is moved to a dead letter queue instead of being requeued.

- Setting `max-retry = 0` (the default) means unlimited retries
- Setting `max-retry = 5` will move a task to the dead letter queue after 5 failed attempts

Tasks in the dead letter queue are stored with their original metadata plus:
- A timestamp in the key name to prevent collisions
- A `failed-time` metadata field indicating when the task was moved to the dead letter queue

Dead-lettered tasks can be inspected using standard GCS tools. They are stored in the workqueue bucket under the `dead-letter/` prefix.

```hcl
module "workqueue" {
  source = "chainguard-dev/common/infra//modules/workqueue"

  # ... other configuration ...

  // Maximum retry limit (5 attempts before moving to dead letter queue)
  max-retry = 5
}
```


<!-- BEGIN_TF_DOCS -->
## Requirements

No requirements.

## Providers

| Name | Version |
|------|---------|
| <a name="provider_google"></a> [google](#provider\_google) | n/a |
| <a name="provider_google-beta"></a> [google-beta](#provider\_google-beta) | n/a |
| <a name="provider_random"></a> [random](#provider\_random) | n/a |

## Modules

| Name | Source | Version |
|------|--------|---------|
| <a name="module_change-trigger-calls-dispatcher"></a> [change-trigger-calls-dispatcher](#module\_change-trigger-calls-dispatcher) | ../authorize-private-service | n/a |
| <a name="module_cron-trigger-calls-dispatcher"></a> [cron-trigger-calls-dispatcher](#module\_cron-trigger-calls-dispatcher) | ../authorize-private-service | n/a |
| <a name="module_dispatcher-calls-target"></a> [dispatcher-calls-target](#module\_dispatcher-calls-target) | ../authorize-private-service | n/a |
| <a name="module_dispatcher-service"></a> [dispatcher-service](#module\_dispatcher-service) | ../regional-go-service | n/a |
| <a name="module_receiver-service"></a> [receiver-service](#module\_receiver-service) | ../regional-go-service | n/a |

## Resources

| Name | Type |
|------|------|
| [google-beta_google_project_service_identity.pubsub](https://registry.terraform.io/providers/hashicorp/google-beta/latest/docs/resources/google_project_service_identity) | resource |
| [google_cloud_scheduler_job.cron](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/cloud_scheduler_job) | resource |
| [google_pubsub_subscription.global-this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/pubsub_subscription) | resource |
| [google_pubsub_topic.global-object-change-notifications](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/pubsub_topic) | resource |
| [google_pubsub_topic_iam_binding.global-gcs-publishes-to-topic](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/pubsub_topic_iam_binding) | resource |
| [google_service_account.change-trigger](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/service_account) | resource |
| [google_service_account.cron-trigger](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/service_account) | resource |
| [google_service_account.dispatcher](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/service_account) | resource |
| [google_service_account.receiver](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/service_account) | resource |
| [google_service_account_iam_binding.allow-pubsub-to-mint-tokens](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/service_account_iam_binding) | resource |
| [google_storage_bucket.global-workqueue](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/storage_bucket) | resource |
| [google_storage_bucket_iam_binding.global-authorize-access](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/storage_bucket_iam_binding) | resource |
| [google_storage_notification.global-object-change-notifications](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/storage_notification) | resource |
| [random_string.bucket_suffix](https://registry.terraform.io/providers/hashicorp/random/latest/docs/resources/string) | resource |
| [random_string.change-trigger](https://registry.terraform.io/providers/hashicorp/random/latest/docs/resources/string) | resource |
| [random_string.cron-trigger](https://registry.terraform.io/providers/hashicorp/random/latest/docs/resources/string) | resource |
| [random_string.dispatcher](https://registry.terraform.io/providers/hashicorp/random/latest/docs/resources/string) | resource |
| [random_string.receiver](https://registry.terraform.io/providers/hashicorp/random/latest/docs/resources/string) | resource |
| [google_storage_project_service_account.gcs_account](https://registry.terraform.io/providers/hashicorp/google/latest/docs/data-sources/storage_project_service_account) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_concurrent-work"></a> [concurrent-work](#input\_concurrent-work) | The amount of concurrent work to dispatch at a given time. | `number` | n/a | yes |
| <a name="input_deletion_protection"></a> [deletion\_protection](#input\_deletion\_protection) | Whether to enable delete protection for the service. | `bool` | `true` | no |
| <a name="input_labels"></a> [labels](#input\_labels) | Labels to apply to the workqueue resources. | `map(string)` | `{}` | no |
| <a name="input_max-retry"></a> [max-retry](#input\_max-retry) | The maximum number of retry attempts before a task is moved to the dead letter queue. Set this to 0 to have unlimited retries. | `number` | `100` | no |
| <a name="input_max-retry2"></a> [max-retry2](#input\_max-retry2) | The maximum number of retry attempts before a task is moved to the dead letter queue. Default of 0 means unlimited retries. | `number` | `5` | no |
| <a name="input_multi_regional_location"></a> [multi\_regional\_location](#input\_multi\_regional\_location) | The multi-regional location for the global workqueue bucket (e.g., 'US', 'EU', 'ASIA'). Only used when scope='global'. | `string` | `"US"` | no |
| <a name="input_name"></a> [name](#input\_name) | n/a | `string` | n/a | yes |
| <a name="input_notification_channels"></a> [notification\_channels](#input\_notification\_channels) | List of notification channels to alert. | `list(string)` | n/a | yes |
| <a name="input_product"></a> [product](#input\_product) | Product label to apply to the service. | `string` | `"unknown"` | no |
| <a name="input_project_id"></a> [project\_project_id](#input\_project\_id) | n/a | `string` | n/a | yes |
| <a name="input_reconciler-service"></a> [reconciler-service](#input\_reconciler-service) | The name of the reconciler service that the workqueue will dispatch work to. | <pre>object({<br/>    name = string<br/>  })</pre> | n/a | yes |
| <a name="input_regions"></a> [regions](#input\_regions) | A map from region names to a network and subnetwork.  A service will be created in each region configured to egress the specified traffic via the specified subnetwork. | <pre>map(object({<br/>    network = string<br/>    subnet  = string<br/>  }))</pre> | n/a | yes |
| <a name="input_scope"></a> [scope](#input\_scope) | The scope of the workqueue. Must be 'global' for a single multi-regional workqueue. | `string` | `"global"` | no |
| <a name="input_squad"></a> [squad](#input\_squad) | squad label to apply to the service. | `string` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_dispatcher"></a> [dispatcher](#output\_dispatcher) | n/a |
| <a name="output_receiver"></a> [receiver](#output\_receiver) | n/a |
<!-- END_TF_DOCS -->
