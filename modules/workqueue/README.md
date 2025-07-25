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
| <a name="module_collapsible"></a> [collapsible](#module\_collapsible) | ../dashboard/sections/collapsible | n/a |
| <a name="module_cron-trigger-calls-dispatcher"></a> [cron-trigger-calls-dispatcher](#module\_cron-trigger-calls-dispatcher) | ../authorize-private-service | n/a |
| <a name="module_dashboard"></a> [dashboard](#module\_dashboard) | ../dashboard | n/a |
| <a name="module_dead-letter-queue"></a> [dead-letter-queue](#module\_dead-letter-queue) | ../dashboard/widgets/xy | n/a |
| <a name="module_dispatcher-calls-target"></a> [dispatcher-calls-target](#module\_dispatcher-calls-target) | ../authorize-private-service | n/a |
| <a name="module_dispatcher-logs"></a> [dispatcher-logs](#module\_dispatcher-logs) | ../dashboard/sections/logs | n/a |
| <a name="module_dispatcher-service"></a> [dispatcher-service](#module\_dispatcher-service) | ../regional-go-service | n/a |
| <a name="module_layout"></a> [layout](#module\_layout) | ../dashboard/sections/layout | n/a |
| <a name="module_max-attempts"></a> [max-attempts](#module\_max-attempts) | ../dashboard/widgets/xy | n/a |
| <a name="module_percent-deduped"></a> [percent-deduped](#module\_percent-deduped) | ../dashboard/widgets/xy-ratio | n/a |
| <a name="module_process-latency"></a> [process-latency](#module\_process-latency) | ../dashboard/widgets/latency | n/a |
| <a name="module_receiver-logs"></a> [receiver-logs](#module\_receiver-logs) | ../dashboard/sections/logs | n/a |
| <a name="module_receiver-service"></a> [receiver-service](#module\_receiver-service) | ../regional-go-service | n/a |
| <a name="module_wait-latency"></a> [wait-latency](#module\_wait-latency) | ../dashboard/widgets/latency | n/a |
| <a name="module_width"></a> [width](#module\_width) | ../dashboard/sections/width | n/a |
| <a name="module_work-added"></a> [work-added](#module\_work-added) | ../dashboard/widgets/xy | n/a |
| <a name="module_work-in-progress"></a> [work-in-progress](#module\_work-in-progress) | ../dashboard/widgets/xy | n/a |
| <a name="module_work-queued"></a> [work-queued](#module\_work-queued) | ../dashboard/widgets/xy | n/a |

## Resources

| Name | Type |
|------|------|
| [google-beta_google_project_service_identity.pubsub](https://registry.terraform.io/providers/hashicorp/google-beta/latest/docs/resources/google_project_service_identity) | resource |
| [google_cloud_scheduler_job.cron](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/cloud_scheduler_job) | resource |
| [google_pubsub_subscription.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/pubsub_subscription) | resource |
| [google_pubsub_topic.object-change-notifications](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/pubsub_topic) | resource |
| [google_pubsub_topic_iam_binding.gcs-publishes-to-topic](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/pubsub_topic_iam_binding) | resource |
| [google_service_account.change-trigger](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/service_account) | resource |
| [google_service_account.cron-trigger](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/service_account) | resource |
| [google_service_account.dispatcher](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/service_account) | resource |
| [google_service_account.receiver](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/service_account) | resource |
| [google_service_account_iam_binding.allow-pubsub-to-mint-tokens](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/service_account_iam_binding) | resource |
| [google_storage_bucket.workqueue](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/storage_bucket) | resource |
| [google_storage_bucket_iam_binding.authorize-access](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/storage_bucket_iam_binding) | resource |
| [google_storage_notification.object-change-notifications](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/storage_notification) | resource |
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
| <a name="input_max-retry"></a> [max-retry](#input\_max-retry) | The maximum number of retry attempts before a task is moved to the dead letter queue. Default of 0 means unlimited retries. | `number` | `0` | no |
| <a name="input_name"></a> [name](#input\_name) | n/a | `string` | n/a | yes |
| <a name="input_notification_channels"></a> [notification\_channels](#input\_notification\_channels) | List of notification channels to alert. | `list(string)` | n/a | yes |
| <a name="input_product"></a> [product](#input\_product) | Product label to apply to the service. | `string` | `"unknown"` | no |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | n/a | `string` | n/a | yes |
| <a name="input_reconciler-service"></a> [reconciler-service](#input\_reconciler-service) | The name of the reconciler service that the workqueue will dispatch work to. | <pre>object({<br/>    name = string<br/>  })</pre> | n/a | yes |
| <a name="input_regions"></a> [regions](#input\_regions) | A map from region names to a network and subnetwork.  A service will be created in each region configured to egress the specified traffic via the specified subnetwork. | <pre>map(object({<br/>    network = string<br/>    subnet  = string<br/>  }))</pre> | n/a | yes |
| <a name="input_squad"></a> [squad](#input\_squad) | squad label to apply to the service. | `string` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_receiver"></a> [receiver](#output\_receiver) | n/a |
<!-- END_TF_DOCS -->
