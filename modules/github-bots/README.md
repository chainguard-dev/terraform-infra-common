# `github-bots`

This module has scaffolding for event-driven GitHub bots. This integrates with [`github-events`](../github-events/) to receive events, and provides SDK methods to interact with GitHub resources. The Terraform module creates a service account for the bot, and deploys the bot as a regional service.

Out-of-the-box bots include:

- [`dnm`](./dnm/): A bot that adds or removes a `blocking/dnm` label on pull requests if the title contains the text "do not merge".
- [`blocker`](./blocker/): A bot that passes or fails a GitHub Check Run based on the presence of a `blocking/*` label on a pull request.
  - this check can be used to block merges in GitHub.

```hcl
// ... networking and cloudevent-broker modules...

module "github-events" {
  source = "./modules/github-events"

  project_id = var.project_id
  name       = "github-events"
  regions    = module.networking.regional-networks
  ingress    = module.cloudevent-broker.ingress

  // Which user is allowed to populate webhook secret values.
  secret_version_adder = "user:you@company.biz"
}

module "bots" {
  source = "./modules/github-bots"
  for_each = {
    "dnm"     = "dev.chainguard.github.pull_request",
    "blocker" = "dev.chainguard.github.pull_request",
  }

  project_id = var.project_id
  regions    = module.networking.regional-networks
  broker     = module.cloudevent-broker.broker

  name         = each.key
  github-event = each.value
  source_code = {
    importpath  = "./${each.key}"
  }
}


module "my-custom-bot" {
  source = "./modules/github-bots"

  project_id = var.project_id
  regions    = module.networking.regional-networks
  broker     = module.cloudevent-broker.broker

  name         = "my-custom-bot"
  github-event = "dev.chainguard.github.pull_request"
  source_code = {
    importpath  = "./cmd/custom/bot"
    working_dir = path.module
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
| <a name="module_cloudevent-trigger"></a> [cloudevent-trigger](#module\_cloudevent-trigger) | chainguard-dev/common/infra//modules/cloudevent-trigger | n/a |
| <a name="module_service"></a> [service](#module\_service) | chainguard-dev/common/infra//modules/regional-go-service | n/a |

## Resources

| Name | Type |
|------|------|
| [google_service_account.sa](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/service_account) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_broker"></a> [broker](#input\_broker) | A map from each of the input region names to the name of the Broker topic in that region. | `map(string)` | n/a | yes |
| <a name="input_github-event"></a> [github-event](#input\_github-event) | The GitHub event type to subscribe to. | `string` | n/a | yes |
| <a name="input_name"></a> [name](#input\_name) | The name of the bot. | `string` | n/a | yes |
| <a name="input_notification_channels"></a> [notification\_channels](#input\_notification\_channels) | List of notification channels to alert. | `list(string)` | n/a | yes |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | Project ID to create resources in. | `string` | n/a | yes |
| <a name="input_regions"></a> [regions](#input\_regions) | A map from region names to a network and subnetwork. | <pre>map(object({<br>    network = string<br>    subnet  = string<br>  }))</pre> | n/a | yes |
| <a name="input_source_code"></a> [source\_code](#input\_source\_code) | The source code for the bot. | <pre>object({<br>    working_dir = string<br>    importpath  = string<br>  })</pre> | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_serviceaccount-email"></a> [serviceaccount-email](#output\_serviceaccount-email) | The ID of the service account for the bot. |
| <a name="output_serviceaccount-id"></a> [serviceaccount-id](#output\_serviceaccount-id) | The ID of the service account for the bot. |
<!-- END_TF_DOCS -->
