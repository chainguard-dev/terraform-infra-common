# `regional-go-service`

This module provisions a regionalizied Go Cloud Run service. The Go code is
built and signed using the `ko` and `cosign` providers. The simplest example
service can be seen here:

```hcl
// Create a network with several regional subnets
module "networking" {
  source = "chainguard-dev/common/infra//modules/networking"

  name       = "my-networking"
  project_id = var.project_id
  regions    = [...]
}

// Create the a service account.
resource "google_service_account" "foo" {
  project = var.project_id

  account_id   = "foo-svac"
  display_name = "foo Service Account"
  description  = "Dedicated service account for the foo service principal."
}

module "foo-service" {
  source = "chainguard-dev/common/infra//modules/regional-go-service"

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
    }
  }

  // Add your notification channels for alerting (required)
  notification_channels = []
}
```

The module is intended to encapsulate Chainguard best practices around deploying
Cloud Run services including:

- More secure default for ingress
- More secure default for egress
- Intentionally not exposing a `uri` output (use
  [`authorize-private-service`](../authorize-private-service/README.md))
- Requiring a service-account name to run as (so as not to use the default
  compute service account!)
- Running an `otel-collector` sidecar container that can collect and publish
  telemetry data from out services (for use with the dashboard modules).

For the most part, we have tried to expose a roughly compatible shape to the
cloud run v2 service itself, with two primary changes:

1. Instead of an `image` string we take a `source` object to feed to `ko_build`,
2. In addition to `env` we support `regional-env`, where the value is a map from
   region to regional value. This can be used to pass different environment
   values to services based on the region they are running in (e.g.
   `cloudevent-broker` ingress endpoint or another regionalized service's
   localized URI).

<!-- BEGIN_TF_DOCS -->
## Requirements

No requirements.

## Providers

| Name | Version |
|------|---------|
| <a name="provider_cosign"></a> [cosign](#provider\_cosign) | n/a |
| <a name="provider_ko"></a> [ko](#provider\_ko) | n/a |

## Modules

| Name | Source | Version |
|------|--------|---------|
| <a name="module_this"></a> [this](#module\_this) | ../regional-service | n/a |

## Resources

| Name | Type |
|------|------|
| [cosign_sign.this](https://registry.terraform.io/providers/chainguard-dev/cosign/latest/docs/resources/sign) | resource |
| [ko_build.this](https://registry.terraform.io/providers/ko-build/ko/latest/docs/resources/build) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_containers"></a> [containers](#input\_containers) | The containers to run in the service.  Each container will be run in each region. | <pre>map(object({<br/>    source = object({<br/>      base_image  = optional(string, "cgr.dev/chainguard/static:latest-glibc@sha256:48278935856fba0e9fac80365ae9a5b33297f7e5682c2dcb86ecfe5eb6878972")<br/>      working_dir = string<br/>      importpath  = string<br/>      env         = optional(list(string), [])<br/>    })<br/>    args = optional(list(string), [])<br/>    ports = optional(list(object({<br/>      name           = optional(string, "http1")<br/>      container_port = number<br/>    })), [])<br/>    resources = optional(<br/>      object(<br/>        {<br/>          limits = optional(object(<br/>            {<br/>              cpu    = string<br/>              memory = string<br/>            }<br/>          ), null)<br/>          cpu_idle          = optional(bool, true)<br/>          startup_cpu_boost = optional(bool, true)<br/>        }<br/>      ),<br/>      {<br/>        cpu_idle = true<br/>      }<br/>    )<br/>    env = optional(list(object({<br/>      name  = string<br/>      value = optional(string)<br/>      value_source = optional(object({<br/>        secret_key_ref = object({<br/>          secret  = string<br/>          version = string<br/>        })<br/>      }), null)<br/>    })), [])<br/>    regional-env = optional(list(object({<br/>      name  = string<br/>      value = map(string)<br/>    })), [])<br/>    volume_mounts = optional(list(object({<br/>      name       = string<br/>      mount_path = string<br/>    })), [])<br/>    startup_probe = optional(object({<br/>      initial_delay_seconds = optional(number)<br/>      // GCP Terraform provider defaults differ from Cloud Run defaults.<br/>      // See https://cloud.google.com/run/docs/configuring/healthchecks#tcp-startup-probe<br/>      period_seconds    = optional(number, 240)<br/>      timeout_seconds   = optional(number, 240)<br/>      failure_threshold = optional(number, 1)<br/>      http_get = optional(object({<br/>        path = string<br/>        port = optional(number)<br/>      }), null)<br/>      tcp_socket = optional(object({<br/>        port = optional(number)<br/>      }), null)<br/>      grpc = optional(object({<br/>        service = optional(string)<br/>        port    = optional(number)<br/>      }), null)<br/>    }))<br/>    liveness_probe = optional(object({<br/>      initial_delay_seconds = optional(number)<br/>      // GCP Terraform provider defaults differ from Cloud Run defaults.<br/>      // See https://cloud.google.com/run/docs/configuring/healthchecks#tcp-startup-probe<br/>      period_seconds    = optional(number, 240)<br/>      timeout_seconds   = optional(number, 240)<br/>      failure_threshold = optional(number, 1)<br/>      http_get = optional(object({<br/>        path = string<br/>        port = optional(number)<br/>      }), null)<br/>      tcp_socket = optional(object({<br/>        port = optional(number)<br/>      }), null)<br/>      grpc = optional(object({<br/>        service = optional(string)<br/>        port    = optional(number)<br/>      }), null)<br/>    }))<br/>  }))</pre> | n/a | yes |
| <a name="input_deletion_protection"></a> [deletion\_protection](#input\_deletion\_protection) | Whether to enable delete protection for the service. | `bool` | `true` | no |
| <a name="input_egress"></a> [egress](#input\_egress) | Which type of egress traffic to send through the VPC.<br/><br/>- ALL\_TRAFFIC sends all traffic through regional VPC network. This should be used if service is not expected to egress to the Internet.<br/>- PRIVATE\_RANGES\_ONLY sends only traffic to private IP addresses through regional VPC network | `string` | `"ALL_TRAFFIC"` | no |
| <a name="input_enable_profiler"></a> [enable\_profiler](#input\_enable\_profiler) | Enable cloud profiler. | `bool` | `false` | no |
| <a name="input_execution_environment"></a> [execution\_environment](#input\_execution\_environment) | The execution environment for the service | `string` | `"EXECUTION_ENVIRONMENT_GEN1"` | no |
| <a name="input_ingress"></a> [ingress](#input\_ingress) | Which type of ingress traffic to accept for the service.<br/><br/>- INGRESS\_TRAFFIC\_ALL accepts all traffic, enabling the public .run.app URL for the service<br/>- INGRESS\_TRAFFIC\_INTERNAL\_LOAD\_BALANCER accepts traffic only from a load balancer<br/>- INGRESS\_TRAFFIC\_INTERNAL\_ONLY accepts internal traffic only | `string` | `"INGRESS_TRAFFIC_INTERNAL_ONLY"` | no |
| <a name="input_labels"></a> [labels](#input\_labels) | Labels to apply to the service. | `map(string)` | `{}` | no |
| <a name="input_name"></a> [name](#input\_name) | n/a | `string` | n/a | yes |
| <a name="input_notification_channels"></a> [notification\_channels](#input\_notification\_channels) | List of notification channels to alert. | `list(string)` | n/a | yes |
| <a name="input_otel_collector_image"></a> [otel\_collector\_image](#input\_otel\_collector\_image) | The otel collector image to use as a base. Must be on gcr.io or dockerhub. | `string` | `"chainguard/opentelemetry-collector-contrib:latest"` | no |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | n/a | `string` | n/a | yes |
| <a name="input_regional-volumes"></a> [regional-volumes](#input\_regional-volumes) | The volumes to make available to the containers in the service for mounting. | <pre>list(object({<br/>    name = string<br/>    gcs = optional(map(object({<br/>      bucket    = string<br/>      read_only = optional(bool, true)<br/>    })), {})<br/>    nfs = optional(map(object({<br/>      server    = string<br/>      path      = string<br/>      read_only = optional(bool, true)<br/>    })), {})<br/>  }))</pre> | `[]` | no |
| <a name="input_regions"></a> [regions](#input\_regions) | A map from region names to a network and subnetwork.  A service will be created in each region configured to egress the specified traffic via the specified subnetwork. | <pre>map(object({<br/>    network = string<br/>    subnet  = string<br/>  }))</pre> | n/a | yes |
| <a name="input_request_timeout_seconds"></a> [request\_timeout\_seconds](#input\_request\_timeout\_seconds) | The timeout for requests to the service, in seconds. | `number` | `300` | no |
| <a name="input_require_squad"></a> [require\_squad](#input\_require\_squad) | Whether to require squad variable to be specified | `bool` | `true` | no |
| <a name="input_scaling"></a> [scaling](#input\_scaling) | The scaling configuration for the service. | <pre>object({<br/>    min_instances                    = optional(number, 0)<br/>    max_instances                    = optional(number, 100)<br/>    max_instance_request_concurrency = optional(number)<br/>  })</pre> | `{}` | no |
| <a name="input_service_account"></a> [service\_account](#input\_service\_account) | The service account as which to run the service. | `string` | n/a | yes |
| <a name="input_squad"></a> [squad](#input\_squad) | squad label to apply to the service. | `string` | `""` | no |
| <a name="input_volumes"></a> [volumes](#input\_volumes) | The volumes to make available to the containers in the service for mounting. | <pre>list(object({<br/>    name = string<br/>    empty_dir = optional(object({<br/>      medium     = optional(string, "MEMORY")<br/>      size_limit = optional(string, "2G")<br/>    }))<br/>    secret = optional(object({<br/>      secret = string<br/>      items = list(object({<br/>        version = string<br/>        path    = string<br/>      }))<br/>    }))<br/>  }))</pre> | `[]` | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_locations"></a> [locations](#output\_locations) | n/a |
| <a name="output_names"></a> [names](#output\_names) | n/a |
| <a name="output_uris"></a> [uris](#output\_uris) | n/a |
<!-- END_TF_DOCS -->
