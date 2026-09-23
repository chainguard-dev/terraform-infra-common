# `github-bots`

This Terraform module deploys a regional GitHub bot service and subscribes it to a GitHub event type through a CloudEvents broker. It creates a service account unless you provide `service_account_email`, and it adds a Cloud Monitoring dashboard.

Supply the project, region network settings, broker topics, container source, GitHub event type, and notification channels. See [the module inputs](./variables.tf) for the current contract and [the module implementation](./main.tf) for the resources it creates. The SDK under [`sdk/`](./sdk/) contains helpers for bot handlers.

<!-- BEGIN_TF_DOCS -->
## Requirements

No requirements.

## Providers

| Name | Version |
| ---- | ------- |
| <a name="provider_google"></a> [google](#provider\_google) | n/a |

## Modules

| Name | Source | Version |
| ---- | ------ | ------- |
| <a name="module_cloudevent-trigger"></a> [cloudevent-trigger](#module\_cloudevent-trigger) | ../cloudevent-trigger | n/a |
| <a name="module_dashboard"></a> [dashboard](#module\_dashboard) | ../dashboard/cloudevent-receiver | n/a |
| <a name="module_service"></a> [service](#module\_service) | ../regional-go-service | n/a |

## Resources

| Name | Type |
| ---- | ---- |
| [google_service_account.sa](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/service_account) | resource |

## Inputs

| Name | Description | Type | Default | Required |
| ---- | ----------- | ---- | ------- | :------: |
| <a name="input_broker"></a> [broker](#input\_broker) | A map from each of the input region names to the name of the Broker topic in that region. | `map(string)` | n/a | yes |
| <a name="input_containers"></a> [containers](#input\_containers) | The containers to run in the service.  Each container will be run in each region. | <pre>map(object({<br/>    source = object({<br/>      base_image  = optional(string, "cgr.dev/chainguard/static:latest-glibc@sha256:bf639cba19ba56329e6907ac26a7afcdde57a80b6aa66d5100da6883196e6b82")<br/>      working_dir = string<br/>      importpath  = string<br/>    })<br/>    args = optional(list(string), [])<br/>    ports = optional(list(object({<br/>      name           = optional(string, "http1")<br/>      container_port = optional(number, 8080)<br/>    })), [])<br/>    resources = optional(<br/>      object(<br/>        {<br/>          limits = optional(object(<br/>            {<br/>              cpu    = string<br/>              memory = string<br/>            }<br/>          ), null)<br/>          cpu_idle          = optional(bool, true)<br/>          startup_cpu_boost = optional(bool, true)<br/>        }<br/>      ),<br/>      {<br/>        cpu_idle = true<br/>      }<br/>    )<br/>    env = optional(list(object({<br/>      name  = string<br/>      value = optional(string)<br/>      value_source = optional(object({<br/>        secret_key_ref = object({<br/>          secret  = string<br/>          version = string<br/>        })<br/>      }), null)<br/>    })), [])<br/>    regional-env = optional(list(object({<br/>      name  = string<br/>      value = map(string)<br/>    })), [])<br/>    volume_mounts = optional(list(object({<br/>      name       = string<br/>      mount_path = string<br/>    })), [])<br/>  }))</pre> | n/a | yes |
| <a name="input_deletion_protection"></a> [deletion\_protection](#input\_deletion\_protection) | Whether to enable delete protection for the service. | `bool` | `true` | no |
| <a name="input_enable_observability_iam"></a> [enable\_observability\_iam](#input\_enable\_observability\_iam) | Whether this module grants the service account the observability roles (monitoring.metricWriter, cloudtrace.agent, cloudprofiler.agent) on the project. Set false when the caller manages these grants itself, e.g. a service account shared across multiple services, where per-service grants would create overlapping non-authoritative IAM members that revoke each other on destroy. | `bool` | `true` | no |
| <a name="input_enable_profiler"></a> [enable\_profiler](#input\_enable\_profiler) | Enable cloud profiler. | `bool` | `false` | no |
| <a name="input_extra_filter"></a> [extra\_filter](#input\_extra\_filter) | Optional additional filters to include. | `map(string)` | `{}` | no |
| <a name="input_extra_filter_has_attributes"></a> [extra\_filter\_has\_attributes](#input\_extra\_filter\_has\_attributes) | Optional additional attributes to check for presence. | `list(string)` | `[]` | no |
| <a name="input_extra_filter_not_has_attributes"></a> [extra\_filter\_not\_has\_attributes](#input\_extra\_filter\_not\_has\_attributes) | Optional additional prefixes to check for presence. | `list(string)` | `[]` | no |
| <a name="input_extra_filter_prefix"></a> [extra\_filter\_prefix](#input\_extra\_filter\_prefix) | Optional additional prefixes for filtering events. | `map(string)` | `{}` | no |
| <a name="input_github-event"></a> [github-event](#input\_github-event) | The GitHub event type to subscribe to. | `string` | n/a | yes |
| <a name="input_labels"></a> [labels](#input\_labels) | Labels to apply to the service. | `map(string)` | `{}` | no |
| <a name="input_launch_stage"></a> [launch\_stage](#input\_launch\_stage) | The launch stage of the Cloud Run service (e.g. BETA to leverage features like disk volumes). | `string` | `"GA"` | no |
| <a name="input_name"></a> [name](#input\_name) | The name of the bot. | `string` | n/a | yes |
| <a name="input_notification_channels"></a> [notification\_channels](#input\_notification\_channels) | List of notification channels to alert. | `list(string)` | n/a | yes |
| <a name="input_observability_role"></a> [observability\_role](#input\_observability\_role) | Fully-qualified id of a single role (e.g. from the observability-role module) to grant the service account in place of the three built-in observability roles (monitoring.metricWriter, cloudtrace.agent, cloudprofiler.agent). Collapsing to one role keeps large projects under the 1,500-member IAM policy limit. | `string` | `null` | no |
| <a name="input_product"></a> [product](#input\_product) | Product label to apply to the service. | `string` | `"unknown"` | no |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | Project ID to create resources in. | `string` | n/a | yes |
| <a name="input_raw_filter"></a> [raw\_filter](#input\_raw\_filter) | Raw PubSub filter to apply, ignores other variables. https://cloud.google.com/pubsub/docs/subscription-message-filter#filtering_syntax | `string` | `""` | no |
| <a name="input_regions"></a> [regions](#input\_regions) | A map from region names to a network and subnetwork. | <pre>map(object({<br/>    network = string<br/>    subnet  = string<br/>  }))</pre> | n/a | yes |
| <a name="input_resource_manager_tags"></a> [resource\_manager\_tags](#input\_resource\_manager\_tags) | Resource Manager tags forwarded to this module's taggable resources, as tagKeys/<id> => tagValues/<id>. | `map(string)` | `{}` | no |
| <a name="input_service_account_email"></a> [service\_account\_email](#input\_service\_account\_email) | The email of the service account being authorized to invoke the private Cloud Run service. If empty, a service account will be created and used. | `string` | `""` | no |
| <a name="input_team"></a> [team](#input\_team) | Team label to apply to resources (replaces deprecated 'squad'). | `string` | n/a | yes |

## Outputs

| Name | Description |
| ---- | ----------- |
| <a name="output_json"></a> [json](#output\_json) | n/a |
| <a name="output_serviceaccount-email"></a> [serviceaccount-email](#output\_serviceaccount-email) | The email of the service account for the bot. |
| <a name="output_serviceaccount-id"></a> [serviceaccount-id](#output\_serviceaccount-id) | The ID of the service account for the bot. |
<!-- END_TF_DOCS -->
