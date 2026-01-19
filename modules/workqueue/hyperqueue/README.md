# `hyperqueue`

This module provisions a "sharded" workqueue.  Behind the scenes, this
provisions N workqueues and stands up a unified workqueue API endpoint that
distributes work consistently over those N shards.  Consistently meaning the
same key will always route to the same shard, which is important for preserving
our concurrency semantics and retaining the ability to fetch key status.

## When should I use the hyperqueue?

The hyperqueue was designed to handle a particular scenario where the workqueue
is dealing with an extraordinarily high volume of keys that each process in a
relatively short amount of time (a torrent, not a trickle).

For example, when we load tested processing Wolfi APKs (300k keys) the latency
to Enumerate our key space and feed work into the system spiked to O(60s), but
each key was only taking O(5-10s) to process.  This meant that we effectively
could not keep the workqueue's concurrency target saturated.  However, when
spread over 5 shards, the Enumerate latency dropped to the point where we could
keep up.

The two metrics to pay attention to from the standard workqueue dashboard are:
1. `Enumerate latency (p95)`, and
2. `Time to completion (p95 by priority)`

The best way to pick a sharding factor is a load test, and the goal should be
for the former to be as low as possible, but ideally close to or lower than the
latter.  If there is a high `Amount of work queued` but we are not keeping the
`Amount of work in progress` at the concurrency target, then the shard factors
_likely_ need adjustment (check the latencies above!).

# Warning: changing the sharding factor

A notable danger of this approach is that changing the sharding factor will mean
that keys will be reassigned and until the key space is flushed and requeued
that keys may exist on two different shards (pre- and post-) and violate the
concurrency properties of the workqueue.  The safest rollout option for
resharding is:
1. **Pause** - stop queuing new work, sharding will change mappings
2. **Drain / Flush** - either wait for work to complete or forcibly delete
3. **Reshard** - we can roll this out during the drain, so long as nothing is calling enqueue
4. **Unpause** - once the queues are empty
5. **Resync** - if we forcibly drained, then forcing a resync ensures that all keys are reconciled to a good state


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
| <a name="module_hyperqueue-calls-receiver"></a> [hyperqueue-calls-receiver](#module\_hyperqueue-calls-receiver) | ../../authorize-private-service | n/a |
| <a name="module_hyperqueue-service"></a> [hyperqueue-service](#module\_hyperqueue-service) | ../../regional-go-service | n/a |
| <a name="module_workqueue"></a> [workqueue](#module\_workqueue) | ../ | n/a |

## Resources

| Name | Type |
|------|------|
| [google_service_account.hyperqueue](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/service_account) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_batch-size"></a> [batch-size](#input\_batch-size) | Optional cap on how much work to launch per dispatcher pass. | `number` | `null` | no |
| <a name="input_concurrent-work"></a> [concurrent-work](#input\_concurrent-work) | The amount of concurrent work to dispatch at a given time (distributed across shards). | `number` | n/a | yes |
| <a name="input_deletion_protection"></a> [deletion\_protection](#input\_deletion\_protection) | Whether to enable delete protection for the service. | `bool` | `true` | no |
| <a name="input_enable_dead_letter_alerting"></a> [enable\_dead\_letter\_alerting](#input\_enable\_dead\_letter\_alerting) | Whether to enable alerting for dead-lettered keys. | `bool` | `true` | no |
| <a name="input_labels"></a> [labels](#input\_labels) | Labels to apply to the workqueue resources. | `map(string)` | `{}` | no |
| <a name="input_max-retry"></a> [max-retry](#input\_max-retry) | The maximum number of retry attempts before a task is moved to the dead letter queue. | `number` | `100` | no |
| <a name="input_multi_regional_location"></a> [multi\_regional\_location](#input\_multi\_regional\_location) | The multi-regional location for the workqueue buckets. | `string` | `"US"` | no |
| <a name="input_name"></a> [name](#input\_name) | n/a | `string` | n/a | yes |
| <a name="input_notification_channels"></a> [notification\_channels](#input\_notification\_channels) | List of notification channels to alert. | `list(string)` | n/a | yes |
| <a name="input_product"></a> [product](#input\_product) | Product label to apply to the service. | `string` | `"unknown"` | no |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | n/a | `string` | n/a | yes |
| <a name="input_reconciler-service"></a> [reconciler-service](#input\_reconciler-service) | The name of the reconciler service that the workqueue will dispatch work to. | <pre>object({<br/>    name = string<br/>  })</pre> | n/a | yes |
| <a name="input_regions"></a> [regions](#input\_regions) | A map from region names to a network and subnetwork. | <pre>map(object({<br/>    network = string<br/>    subnet  = string<br/>  }))</pre> | n/a | yes |
| <a name="input_shards"></a> [shards](#input\_shards) | Number of workqueue shards (2-5). Each shard is an independent workqueue. | `number` | `2` | no |
| <a name="input_team"></a> [team](#input\_team) | Team label to apply to resources. | `string` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_receiver"></a> [receiver](#output\_receiver) | The hyperqueue router service (clients queue work here) |
<!-- END_TF_DOCS -->
