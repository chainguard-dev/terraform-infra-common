# `cloudevent-broker`

This module provisions a regionalizied Broker abstraction akin to the Knative
"Broker" concept.  The dual "Trigger" concept is captured by the sibling
`cloudevent-trigger` module.  The intended usage of this module for publishing
events is something like this:

```hcl
// Create a network with several regional subnets
module "networking" {
  source = "chainguard-dev/glue/cloudrun//networking"

  name       = "my-networking"
  project_id = var.project_id
  regions    = [...]
}

// Create the Broker abstraction.
module "cloudevent-broker" {
  source = "chainguard-dev/glue/cloudrun//cloudevent-broker"

  name       = "my-broker"
  project_id = var.project_id
  regions    = module.networking.regional-networks
}

// Authorize the "foo" service account to publish events.
module "foo-emits-events" {
  for_each = module.networking.regional-networks

  source = "chainguard-dev/glue/cloudrun//authorize-private-service"

  project_id = var.project_id
  region     = each.key
  name       = module.cloudevent-broker.ingress.name

  service-account = google_service_account.foo.email
}

// Run a cloud run service as the "foo" service account, and pass in the address
// of the regional ingress endpoint.
module "foo-service" {
  source = "chainguard-dev/glue/cloudrun//regional-go-service"

  project_id = var.project_id
  name       = "foo"
  regions    = module.networking.regional-networks

  service_account = google_service_account.foo.email
  containers = {
    "foo" = {
      source = {
        working_dir = path.module
        importpath  = "./cmd/foo"
      }
      ports = [{ container_port = 8080 }]
      regional-env = [{
        name  = "PUBSUB_TOPIC"
        value = { for k, v in module.foo-emits-events : k => v.uri }
      }]
    }
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

## Modules

| Name | Source | Version |
|------|--------|---------|
| <a name="module_http"></a> [http](#module\_http) | ../dashboard/sections/http | n/a |
| <a name="module_layout"></a> [layout](#module\_layout) | ../dashboard/sections/layout | n/a |
| <a name="module_logs"></a> [logs](#module\_logs) | ../dashboard/sections/logs | n/a |
| <a name="module_resources"></a> [resources](#module\_resources) | ../dashboard/sections/resources | n/a |
| <a name="module_this"></a> [this](#module\_this) | ../regional-go-service | n/a |
| <a name="module_topic"></a> [topic](#module\_topic) | ../dashboard/sections/topic | n/a |
| <a name="module_width"></a> [width](#module\_width) | ../dashboard/sections/width | n/a |

## Resources

| Name | Type |
|------|------|
| [google_monitoring_dashboard.dashboard](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/monitoring_dashboard) | resource |
| [google_pubsub_topic.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/pubsub_topic) | resource |
| [google_pubsub_topic_iam_binding.ingress-publishes-events](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/pubsub_topic_iam_binding) | resource |
| [google_service_account.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/service_account) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_name"></a> [name](#input\_name) | n/a | `string` | n/a | yes |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | n/a | `string` | n/a | yes |
| <a name="input_regions"></a> [regions](#input\_regions) | A map from region names to a network and subnetwork.  A pub/sub topic and ingress service (publishing to the respective topic) will be created in each region, with the ingress service configured to egress all traffic via the specified subnetwork. | <pre>map(object({<br>    network = string<br>    subnet  = string<br>  }))</pre> | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_broker"></a> [broker](#output\_broker) | A map from each of the input region names to the name of the Broker topic in each region.  These broker names are intended for use with the cloudevent-trigger module's broker input. |
| <a name="output_ingress"></a> [ingress](#output\_ingress) | An object holding the name of the ingress service, which can be used to authorize callers to publish cloud events. |
<!-- END_TF_DOCS -->
