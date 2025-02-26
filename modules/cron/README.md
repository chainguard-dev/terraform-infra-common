# `cron`

This module provisions a cron job running a custom Go application in one region.

A cron job can be defined as a simple Go program, with as little code as:

```go
import "log"

func main() {
    log.Println("hello")
}
```

> See our [example](./example/).

The Go code is built and signed using the `ko` and `cosign` providers. The
simplest example service can be seen here:

```terraform
module "cron" {
  source  = "chainguard-dev/common/infra//modules/cron"

  name       = "example"
  project_id = var.project_id
  region     = "us-central1" # Optional, defaults to "us-east4".
  schedule   = "*/8 * * * *"  # Every 8 minutes.

  importpath  = "github.com/chainguard-dev/terraform-infra-common/cron/example"
  working_dir = path.module
}
```

> See our [example](./example/).

## Passing additional configuration

You can pass additional configuration to your custom cron jobs via environment
variables passed to the application. These can be specified in the module:

```terraform
  env = {
    "FOO" : "bar"
  }
```

> See our [example](./example/).

or as from a secret in Google Secret Manager:

```terraform
  secret_env = {
    "FOO" : "secret_name_in_secret_manager"
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
| <a name="provider_ko"></a> [ko](#provider\_ko) | n/a |
| <a name="provider_null"></a> [null](#provider\_null) | n/a |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [google-beta_google_cloud_run_v2_job.job](https://registry.terraform.io/providers/hashicorp/google-beta/latest/docs/resources/google_cloud_run_v2_job) | resource |
| [google_cloud_run_v2_job_iam_binding.authorize-calls](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/cloud_run_v2_job_iam_binding) | resource |
| [google_cloud_scheduler_job.cron](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/cloud_scheduler_job) | resource |
| [google_monitoring_alert_policy.success](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/monitoring_alert_policy) | resource |
| [google_project_iam_member.authorize-list](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/project_iam_member) | resource |
| [google_project_iam_member.metrics-writer](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/project_iam_member) | resource |
| [google_project_iam_member.profiler-writer](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/project_iam_member) | resource |
| [google_project_iam_member.trace-writer](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/project_iam_member) | resource |
| [google_project_service.cloud_run_api](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/project_service) | resource |
| [google_project_service.cloudscheduler](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/project_service) | resource |
| [google_service_account.delivery](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/service_account) | resource |
| [ko_build.image](https://registry.terraform.io/providers/ko-build/ko/latest/docs/resources/build) | resource |
| [null_resource.exec](https://registry.terraform.io/providers/hashicorp/null/latest/docs/resources/resource) | resource |
| [google_client_config.default](https://registry.terraform.io/providers/hashicorp/google/latest/docs/data-sources/client_config) | data source |
| [google_client_openid_userinfo.me](https://registry.terraform.io/providers/hashicorp/google/latest/docs/data-sources/client_openid_userinfo) | data source |
| [google_project.project](https://registry.terraform.io/providers/hashicorp/google/latest/docs/data-sources/project) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_base_image"></a> [base\_image](#input\_base\_image) | The base image that will be used to build the container image. | `string` | `"cgr.dev/chainguard/static:latest-glibc"` | no |
| <a name="input_cpu"></a> [cpu](#input\_cpu) | The CPU limit for the job. | `string` | `"1000m"` | no |
| <a name="input_deletion_protection"></a> [deletion\_protection](#input\_deletion\_protection) | Whether to enable delete protection for the service. | `bool` | `true` | no |
| <a name="input_enable_otel_sidecar"></a> [enable\_otel\_sidecar](#input\_enable\_otel\_sidecar) | Enable otel sidecar for metrics | `bool` | `false` | no |
| <a name="input_env"></a> [env](#input\_env) | A map of custom environment variables (e.g. key=value) | `map` | `{}` | no |
| <a name="input_exec"></a> [exec](#input\_exec) | Whether to execute job on modify. | `bool` | `false` | no |
| <a name="input_execution_environment"></a> [execution\_environment](#input\_execution\_environment) | The execution environment to use for the job. | `string` | `""` | no |
| <a name="input_importpath"></a> [importpath](#input\_importpath) | The import path that contains the cron application. Leave empty to run the unmodified base image as the application: for example, when running an `apko`-built image. This works by skipping the `ko` build and just use the base image directly in the cron job. A digest must be specified in this case. | `string` | `""` | no |
| <a name="input_invokers"></a> [invokers](#input\_invokers) | List of iam members invoker perimssions to invoke the job. | `list(string)` | `[]` | no |
| <a name="input_ko_build_env"></a> [ko\_build\_env](#input\_ko\_build\_env) | A list of custom environment variables to pass to the ko build. | `list(string)` | `[]` | no |
| <a name="input_labels"></a> [labels](#input\_labels) | Labels to apply to the job. | `map(string)` | `{}` | no |
| <a name="input_max_retries"></a> [max\_retries](#input\_max\_retries) | The maximum number of times to retry the job. | `number` | `3` | no |
| <a name="input_memory"></a> [memory](#input\_memory) | The memory limit for the job. | `string` | `"512Mi"` | no |
| <a name="input_name"></a> [name](#input\_name) | Name to prefix to created resources. | `any` | n/a | yes |
| <a name="input_notification_channels"></a> [notification\_channels](#input\_notification\_channels) | List of notification channels to alert. | `list(string)` | n/a | yes |
| <a name="input_otel_collector_image"></a> [otel\_collector\_image](#input\_otel\_collector\_image) | The otel collector image to use as a base. Must be on gcr.io or dockerhub. | `string` | `"chainguard/opentelemetry-collector-contrib:latest"` | no |
| <a name="input_parallelism"></a> [parallelism](#input\_parallelism) | The number of parallel jobs to run. Must be <= task\_count | `number` | `1` | no |
| <a name="input_paused"></a> [paused](#input\_paused) | Whether the cron scheduler is paused or not. | `bool` | `false` | no |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | The project that will host the cron job. | `string` | n/a | yes |
| <a name="input_region"></a> [region](#input\_region) | The region to run the job. | `string` | `"us-east4"` | no |
| <a name="input_repository"></a> [repository](#input\_repository) | Container repository to publish images to. | `string` | `""` | no |
| <a name="input_require_squad"></a> [require\_squad](#input\_require\_squad) | Whether to require squad variable to be specified | `bool` | `true` | no |
| <a name="input_schedule"></a> [schedule](#input\_schedule) | The cron schedule on which to run the job. | `any` | n/a | yes |
| <a name="input_scheduled_env_overrides"></a> [scheduled\_env\_overrides](#input\_scheduled\_env\_overrides) | List of env object overrides. | <pre>list(object({<br/>    name  = string<br/>    value = string<br/>  }))</pre> | `[]` | no |
| <a name="input_secret_env"></a> [secret\_env](#input\_secret\_env) | A map of secrets to mount as environment variables from Google Secrets Manager (e.g. secret\_key=secret\_name) | `map` | `{}` | no |
| <a name="input_service_account"></a> [service\_account](#input\_service\_account) | The email address of the service account to run the service as, and to invoke the job as. | `string` | n/a | yes |
| <a name="input_squad"></a> [squad](#input\_squad) | squad label to apply to the service. | `string` | `""` | no |
| <a name="input_success_alert_alignment_period_seconds"></a> [success\_alert\_alignment\_period\_seconds](#input\_success\_alert\_alignment\_period\_seconds) | Alignment period for successful completion alert. 0 (default) to not create alert. | `number` | `0` | no |
| <a name="input_task_count"></a> [task\_count](#input\_task\_count) | The number of tasks to run. | `number` | `1` | no |
| <a name="input_timeout"></a> [timeout](#input\_timeout) | The maximum amount of time in seconds to allow the job to run. | `string` | `"600s"` | no |
| <a name="input_volume_mounts"></a> [volume\_mounts](#input\_volume\_mounts) | The volume mounts to mount the volumes to the container in the job. | <pre>list(object({<br/>    name       = string<br/>    mount_path = string<br/>  }))</pre> | `[]` | no |
| <a name="input_volumes"></a> [volumes](#input\_volumes) | The volumes to make available to the container in the job for mounting. | <pre>list(object({<br/>    name = string<br/>    empty_dir = optional(object({<br/>      medium     = optional(string, "MEMORY")<br/>      size_limit = optional(string)<br/>    }))<br/>    secret = optional(object({<br/>      secret = string<br/>      items = list(object({<br/>        version = string<br/>        path    = string<br/>      }))<br/>    }))<br/>    nfs = optional(object({<br/>      server    = string<br/>      path      = string<br/>      read_only = optional(bool, true)<br/>    }))<br/>  }))</pre> | `[]` | no |
| <a name="input_vpc_access"></a> [vpc\_access](#input\_vpc\_access) | The VPC to send egress to. For more information, visit https://cloud.google.com/run/docs/configuring/vpc-direct-vpc | <pre>object({<br/>    # Currently, only one network interface is supported.<br/>    network_interfaces = list(object({<br/>      network    = string<br/>      subnetwork = string<br/>      tags       = optional(list(string))<br/>    }))<br/>    # Egress is one of "PRIVATE_RANGES_ONLY", "ALL_TRAFFIC", or "ALL_PRIVATE_RANGES"<br/>    egress = string<br/>  })</pre> | `null` | no |
| <a name="input_working_dir"></a> [working\_dir](#input\_working\_dir) | The working directory that contains the importpath. | `string` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_id"></a> [id](#output\_id) | n/a |
| <a name="output_name"></a> [name](#output\_name) | n/a |
<!-- END_TF_DOCS -->
