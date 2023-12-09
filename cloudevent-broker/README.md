# `cloudevent-broker`

This module provisions a regionalizied Broker abstraction akin to the Knative
"Broker" concept.  The dual "Trigger" concept is captured by the sibling
`cloudevent-trigger` module.  The intended usage of this module for publishing
events is something like this:

```hcl
// Create the Broker abstraction.
module "cloudevent-broker" {
  source = "chainguard-dev/glue/cloudrun//cloudevent-broker"

  name       = "my-broker"
  project_id = var.project_id
  regions    = local.region-to-networking
}

// Authorize the "foo" service account to publish events.
module "foo-emits-events" {
  for_each = local.region-to-networking

  source = "chainguard-dev/glue/cloudrun//authorize-private-service"

  project_id = var.project_id
  region     = each.key
  name       = module.cloudevent-broker.ingress.name

  service-account = google_service_account.foo.email
}

// Run a cloud run service as the "foo" service account, and pass in the address
// of the regional ingress endpoint.
resource "google_cloud_run_v2_service" "foo-service" {
  for_each = local.region-to-networking

  project  = var.project_id
  name     = "foo"
  location = each.key

  launch_stage = "BETA"

  template {
    vpc_access {
      network_interfaces {
        network    = each.value.network
        subnetwork = each.value.subnet
      }
      // Egress through VPC so we can talk to the private ingress endpoint.
      egress = "PRIVATE_RANGES_ONLY"
    }

    service_account = google_service_account.foo.email

    containers {
      image = "..."

      // Pass the resolved regional URI to the service in this region.
      env {
        name  = "EVENT_INGRESS_URL"
        value = module.foo-emits-events[each.key].uri
      }
    }
  }
}

// TODO(mattmoor): Put together an example showing how to set up
// local.region-to-networking with the appropriate pieces.
```

<!-- BEGIN_TF_DOCS -->
## Requirements

No requirements.

## Providers

| Name | Version |
|------|---------|
| <a name="provider_cosign"></a> [cosign](#provider\_cosign) | n/a |
| <a name="provider_google"></a> [google](#provider\_google) | n/a |
| <a name="provider_ko"></a> [ko](#provider\_ko) | n/a |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [cosign_sign.this](https://registry.terraform.io/providers/chainguard-dev/cosign/latest/docs/resources/sign) | resource |
| [google_cloud_run_v2_service.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/cloud_run_v2_service) | resource |
| [google_pubsub_topic.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/pubsub_topic) | resource |
| [google_pubsub_topic_iam_binding.ingress-publishes-events](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/pubsub_topic_iam_binding) | resource |
| [google_service_account.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/service_account) | resource |
| [ko_build.this](https://registry.terraform.io/providers/ko-build/ko/latest/docs/resources/build) | resource |

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
