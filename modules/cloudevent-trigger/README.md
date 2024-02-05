# `cloudevent-trigger`

This module provisions regionalizied event-triggered services using a Trigger
abstraction akin to the Knative "Trigger" concept. The dual "Broker" concept is
captured by the sibling `cloudevent-broker` module. The intended usage of this
module for consuming events is something like this:

```hcl
// Create a network with several regional subnets
module "networking" {
  source = "chainguard-dev/common/infra//modules/networking"

  name       = "my-networking"
  project_id = var.project_id
  regions    = [...]
}

// Create the Broker abstraction.
module "cloudevent-broker" {
  source = "chainguard-dev/common/infra//modules/cloudevent-broker"

  name       = "my-broker"
  project_id = var.project_id
  regions    = module.networking.regional-networks
}

module "bar-service" {
  source = "chainguard-dev/common/infra//modules/regional-go-service"

  project_id = var.project_id
  name       = "bar"
  regions    = module.networking.regional-networks

  service_account = google_service_account.bar.email
  containers = {
    "foo" = {
      source = {
        working_dir = path.module
        importpath  = "./cmd/foo"
      }
      ports = [{ container_port = 8080 }]
    }
  }
}

// Set up regionalized triggers to deliver filtered events from our regionalized
// brokers to our regionalized consumer services.
module "cloudevent-trigger" {
  for_each = module.networking.regional-networks

  source = "chainguard-dev/common/infra//modules/cloudevent-trigger"

  name       = "bar"
  project_id = var.project_id
  broker     = module.cloudevent-broker.broker[each.key]
  filter     = { "type" : "dev.chainguard.bar" }

  depends_on = [google_cloud_run_v2_service.fanout-service]
  private-service = {
    region = each.key
    name   = google_cloud_run_v2_service.bar-service[each.key].name
  }
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
| <a name="module_authorize-delivery"></a> [authorize-delivery](#module\_authorize-delivery) | ../authorize-private-service | n/a |

## Resources

| Name | Type |
|------|------|
| [google-beta_google_project_service_identity.pubsub](https://registry.terraform.io/providers/hashicorp/google-beta/latest/docs/resources/google_project_service_identity) | resource |
| [google_pubsub_subscription.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/pubsub_subscription) | resource |
| [google_service_account.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/service_account) | resource |
| [google_service_account_iam_binding.allow-pubsub-to-mint-tokens](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/service_account_iam_binding) | resource |
| [random_string.suffix](https://registry.terraform.io/providers/hashicorp/random/latest/docs/resources/string) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_broker"></a> [broker](#input\_broker) | The name of the pubsub topic we are using as a broker. | `string` | n/a | yes |
| <a name="input_expiration_policy"></a> [expiration\_policy](#input\_expiration\_policy) | The expiration policy for the subscription. | <pre>object({<br>    ttl = optional(string, null)<br>  })</pre> | <pre>{<br>  "ttl": ""<br>}</pre> | no |
| <a name="input_filter"></a> [filter](#input\_filter) | A Knative Trigger-style filter over the cloud event attributes. | `map(string)` | n/a | yes |
| <a name="input_name"></a> [name](#input\_name) | n/a | `string` | n/a | yes |
| <a name="input_private-service"></a> [private-service](#input\_private-service) | The private cloud run service that is subscribing to these events. | <pre>object({<br>    name   = string<br>    region = string<br>  })</pre> | n/a | yes |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | n/a | `string` | n/a | yes |
| <a name="input_retry_policy"></a> [retry\_policy](#input\_retry\_policy) | The retry policy for the subscription. | <pre>object({<br>    minimum_backoff = optional(string, null)<br>    maximum_backoff = optional(string, null)<br>  })</pre> | `null` | no |

## Outputs

No outputs.
<!-- END_TF_DOCS -->
