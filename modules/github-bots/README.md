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
  containers = {
    "bot" = {
      source = {
        importpath  = "./${each.key}"
      }
      env = [
        {
          name  = "FOO"
          value = "BAR"
        }
      ]
    }
  }
}


module "my-custom-bot" {
  source = "./modules/github-bots"

  project_id = var.project_id
  regions    = module.networking.regional-networks
  broker     = module.cloudevent-broker.broker

  name         = "my-custom-bot"
  github-event = "dev.chainguard.github.pull_request"
  containers = {
    "bot" = {
      source = {
        working_dir = path.module
        importpath  = "chainguard.dev/bots/my-custom-bot"
      }
      ports = [{ container_port = 8080 }]
      env = [{
        name  = "LOG_LEVEL"
        value = "info"
      }]
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
| <a name="module_cloudevent-trigger"></a> [cloudevent-trigger](#module\_cloudevent-trigger) | ../cloudevent-trigger | n/a |
| <a name="module_dashboard"></a> [dashboard](#module\_dashboard) | chainguard-dev/common/infra//modules/dashboard/cloudevent-receiver | n/a |
| <a name="module_service"></a> [service](#module\_service) | ../regional-go-service | n/a |

## Resources

| Name | Type |
|------|------|
| [google_service_account.sa](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/service_account) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_broker"></a> [broker](#input\_broker) | A map from each of the input region names to the name of the Broker topic in that region. | `map(string)` | n/a | yes |
| <a name="input_containers"></a> [containers](#input\_containers) | The containers to run in the service.  Each container will be run in each region. | <pre>map(object({<br>    source = object({<br>      base_image  = optional(string, "cgr.dev/chainguard/static:latest-glibc")<br>      working_dir = string<br>      importpath  = string<br>    })<br>    args = optional(list(string), [])<br>    ports = optional(list(object({<br>      name           = optional(string, "http1")<br>      container_port = optional(number, 8080)<br>    })), [])<br>    resources = optional(<br>      object(<br>        {<br>          limits = optional(object(<br>            {<br>              cpu    = string<br>              memory = string<br>            }<br>          ), null)<br>          cpu_idle          = optional(bool, true)<br>          startup_cpu_boost = optional(bool, false)<br>        }<br>      ),<br>      {<br>        cpu_idle = true<br>      }<br>    )<br>    env = optional(list(object({<br>      name  = string<br>      value = optional(string)<br>      value_source = optional(object({<br>        secret_key_ref = object({<br>          secret  = string<br>          version = string<br>        })<br>      }), null)<br>    })), [])<br>    regional-env = optional(list(object({<br>      name  = string<br>      value = map(string)<br>    })), [])<br>    volume_mounts = optional(list(object({<br>      name       = string<br>      mount_path = string<br>    })), [])<br>  }))</pre> | n/a | yes |
| <a name="input_enable_profiler"></a> [enable\_profiler](#input\_enable\_profiler) | Enable cloud profiler. | `bool` | `false` | no |
| <a name="input_extra_filter"></a> [extra\_filter](#input\_extra\_filter) | Optional additional filters to include. | `map(string)` | `{}` | no |
| <a name="input_extra_filter_has_attributes"></a> [extra\_filter\_has\_attributes](#input\_extra\_filter\_has\_attributes) | Optional additional attributes to check for presence. | `list(string)` | `[]` | no |
| <a name="input_extra_filter_not_has_attributes"></a> [extra\_filter\_not\_has\_attributes](#input\_extra\_filter\_not\_has\_attributes) | Optional additional prefixes to check for presence. | `list(string)` | `[]` | no |
| <a name="input_extra_filter_prefix"></a> [extra\_filter\_prefix](#input\_extra\_filter\_prefix) | Optional additional prefixes for filtering events. | `map(string)` | `{}` | no |
| <a name="input_github-event"></a> [github-event](#input\_github-event) | The GitHub event type to subscribe to. | `string` | n/a | yes |
| <a name="input_name"></a> [name](#input\_name) | The name of the bot. | `string` | n/a | yes |
| <a name="input_notification_channels"></a> [notification\_channels](#input\_notification\_channels) | List of notification channels to alert. | `list(string)` | n/a | yes |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | Project ID to create resources in. | `string` | n/a | yes |
| <a name="input_regions"></a> [regions](#input\_regions) | A map from region names to a network and subnetwork. | <pre>map(object({<br>    network = string<br>    subnet  = string<br>  }))</pre> | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_serviceaccount-email"></a> [serviceaccount-email](#output\_serviceaccount-email) | The ID of the service account for the bot. |
| <a name="output_serviceaccount-id"></a> [serviceaccount-id](#output\_serviceaccount-id) | The ID of the service account for the bot. |
<!-- END_TF_DOCS -->
