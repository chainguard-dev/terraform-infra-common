<!-- BEGIN_TF_DOCS -->
## Requirements

No requirements.

## Providers

| Name | Version |
| ---- | ------- |
| <a name="provider_cosign"></a> [cosign](#provider\_cosign) | n/a |
| <a name="provider_google"></a> [google](#provider\_google) | n/a |
| <a name="provider_google-beta"></a> [google-beta](#provider\_google-beta) | n/a |
| <a name="provider_ko"></a> [ko](#provider\_ko) | n/a |

## Modules

| Name | Source | Version |
| ---- | ------ | ------- |
| <a name="module_invoker_name"></a> [invoker\_name](#module\_invoker\_name) | ../limited-concat | n/a |

## Resources

| Name | Type |
| ---- | ---- |
| [cosign_sign.this](https://registry.terraform.io/providers/chainguard-dev/cosign/latest/docs/resources/sign) | resource |
| [google-beta_google_cloud_run_v2_job.this](https://registry.terraform.io/providers/hashicorp/google-beta/latest/docs/resources/google_cloud_run_v2_job) | resource |
| [google_cloud_run_v2_job_iam_binding.authorize-calls](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/cloud_run_v2_job_iam_binding) | resource |
| [google_cloud_scheduler_job.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/cloud_scheduler_job) | resource |
| [google_monitoring_alert_policy.success](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/monitoring_alert_policy) | resource |
| [google_project_iam_member.metrics-writer](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/project_iam_member) | resource |
| [google_project_iam_member.observability](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/project_iam_member) | resource |
| [google_project_iam_member.profiler-writer](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/project_iam_member) | resource |
| [google_project_iam_member.trace-writer](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/project_iam_member) | resource |
| [google_service_account.invoker](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/service_account) | resource |
| [ko_build.this](https://registry.terraform.io/providers/ko-build/ko/latest/docs/resources/build) | resource |

## Inputs

| Name | Description | Type | Default | Required |
| ---- | ----------- | ---- | ------- | :------: |
| <a name="input_containers"></a> [containers](#input\_containers) | The containers to run in each job task. Ports, probes, and cpu\_idle are accepted for type compatibility with regional-go-service but are not used in job tasks. | <pre>map(object({<br/>    source = object({<br/>      base_image  = optional(string, "cgr.dev/chainguard/static:latest-glibc@sha256:60582b2ae6074f641094af0f370d4ab241aab271858a66223dcde7eee9f51638")<br/>      working_dir = string<br/>      importpath  = string<br/>      env         = optional(list(string), [])<br/>    })<br/>    command = optional(list(string), [])<br/>    args    = optional(list(string), [])<br/>    ports = optional(list(object({<br/>      name           = optional(string, "h2c")<br/>      container_port = number<br/>    })), [])<br/>    resources = optional(object({<br/>      limits = optional(object({<br/>        cpu    = string<br/>        memory = string<br/>      }), null)<br/>      cpu_idle          = optional(bool)<br/>      startup_cpu_boost = optional(bool, true)<br/>    }), {})<br/>    env = optional(list(object({<br/>      name  = string<br/>      value = optional(string)<br/>      value_source = optional(object({<br/>        secret_key_ref = object({<br/>          secret  = string<br/>          version = string<br/>        })<br/>      }), null)<br/>    })), [])<br/>    regional-env = optional(list(object({<br/>      name  = string<br/>      value = map(string)<br/>    })), [])<br/>    regional-cpu-idle = optional(map(bool), {})<br/>    volume_mounts = optional(list(object({<br/>      name       = string<br/>      mount_path = string<br/>    })), [])<br/>    startup_probe  = optional(any)<br/>    liveness_probe = optional(any)<br/>  }))</pre> | `{}` | no |
| <a name="input_deletion_protection"></a> [deletion\_protection](#input\_deletion\_protection) | Whether to enable delete protection on the Cloud Run Jobs. | `bool` | `true` | no |
| <a name="input_egress"></a> [egress](#input\_egress) | Which type of egress traffic to route through the VPC. ALL\_TRAFFIC or PRIVATE\_RANGES\_ONLY. | `string` | `"ALL_TRAFFIC"` | no |
| <a name="input_enable_otel_sidecar"></a> [enable\_otel\_sidecar](#input\_enable\_otel\_sidecar) | n/a | `bool` | `true` | no |
| <a name="input_execution_environment"></a> [execution\_environment](#input\_execution\_environment) | n/a | `string` | `"EXECUTION_ENVIRONMENT_GEN2"` | no |
| <a name="input_invokers"></a> [invokers](#input\_invokers) | Additional IAM members granted roles/run.invoker on the job, beyond the dedicated invoker service account. | `list(string)` | `[]` | no |
| <a name="input_labels"></a> [labels](#input\_labels) | Additional labels to apply to all resources. | `map(string)` | `{}` | no |
| <a name="input_launch_stage"></a> [launch\_stage](#input\_launch\_stage) | n/a | `string` | `"GA"` | no |
| <a name="input_max_retries"></a> [max\_retries](#input\_max\_retries) | Maximum number of times a task is retried on failure. 0 means no retries. | `number` | `0` | no |
| <a name="input_name"></a> [name](#input\_name) | n/a | `string` | n/a | yes |
| <a name="input_notification_channels"></a> [notification\_channels](#input\_notification\_channels) | Notification channels for alerts. | `list(string)` | `[]` | no |
| <a name="input_observability_role"></a> [observability\_role](#input\_observability\_role) | Fully-qualified id of a single role (e.g. from the observability-role module) to grant the service account in place of the three built-in observability roles (monitoring.metricWriter, cloudtrace.agent, cloudprofiler.agent). Collapsing to one role keeps large projects under the 1,500-member IAM policy limit. | `string` | `null` | no |
| <a name="input_otel_collector_image"></a> [otel\_collector\_image](#input\_otel\_collector\_image) | The otel collector image to use as a base. Must be on gcr.io or dockerhub. The bundled scrape config enables native histogram scraping by default, which needs opentelemetry-collector-contrib v0.142.0 or later; older collectors reject the config at startup. | `string` | `"chainguard/opentelemetry-collector-contrib:latest"` | no |
| <a name="input_otel_resources"></a> [otel\_resources](#input\_otel\_resources) | Resources to add to the OpenTelemetry resource. | `map(string)` | `{}` | no |
| <a name="input_parallelism"></a> [parallelism](#input\_parallelism) | n/a | `number` | `1` | no |
| <a name="input_product"></a> [product](#input\_product) | Product label to apply to resources. | `string` | `"unknown"` | no |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | n/a | `string` | n/a | yes |
| <a name="input_regional-cronspec"></a> [regional-cronspec](#input\_regional-cronspec) | Per-region cron schedule configuration. Must contain an entry for every key in var.regions. | <pre>map(object({<br/>    schedule  = string<br/>    time_zone = optional(string, "America/New_York")<br/>    paused    = optional(bool, false)<br/>  }))</pre> | n/a | yes |
| <a name="input_regions"></a> [regions](#input\_regions) | A map from region names to a network and subnetwork. A job and scheduler will be created in each region. | <pre>map(object({<br/>    network = optional(string)<br/>    subnet  = optional(string)<br/>  }))</pre> | n/a | yes |
| <a name="input_scrape_native_histograms"></a> [scrape\_native\_histograms](#input\_scrape\_native\_histograms) | Scrape native (exponential) histograms from metrics targets. Requires opentelemetry-collector-contrib v0.142.0 or later. Set to false when pinning otel\_collector\_image to an older collector, which rejects the scrape keys at startup. | `bool` | `true` | no |
| <a name="input_service_account"></a> [service\_account](#input\_service\_account) | The service account as which each job task runs. | `string` | n/a | yes |
| <a name="input_success_alert_alignment_period_seconds"></a> [success\_alert\_alignment\_period\_seconds](#input\_success\_alert\_alignment\_period\_seconds) | Alignment period for successful completion alert. 0 (default) to not create alert. | `number` | `0` | no |
| <a name="input_success_alert_documentation"></a> [success\_alert\_documentation](#input\_success\_alert\_documentation) | Markdown documentation attached to the success-absence alert. Shown in the incident and notification (e.g. a runbook or a Logs Explorer link). Empty (default) attaches none. | `string` | `""` | no |
| <a name="input_success_alert_duration_seconds"></a> [success\_alert\_duration\_seconds](#input\_success\_alert\_duration\_seconds) | How long the absence of successful executions must persist before the alert fires. If not set or 0, defaults to success\_alert\_alignment\_period\_seconds for backward compatibility. | `number` | `0` | no |
| <a name="input_task_count"></a> [task\_count](#input\_task\_count) | n/a | `number` | `1` | no |
| <a name="input_team"></a> [team](#input\_team) | Team label to apply to resources. | `string` | n/a | yes |
| <a name="input_timeout"></a> [timeout](#input\_timeout) | Maximum time allowed for a single task execution. | `string` | `"600s"` | no |
| <a name="input_volumes"></a> [volumes](#input\_volumes) | Volumes to make available to job task containers. | <pre>list(object({<br/>    name = string<br/>    empty_dir = optional(object({<br/>      medium     = optional(string, "MEMORY")<br/>      size_limit = optional(string)<br/>    }))<br/>    secret = optional(object({<br/>      secret = string<br/>      items = list(object({<br/>        version = string<br/>        path    = string<br/>      }))<br/>    }))<br/>    nfs = optional(object({<br/>      server    = string<br/>      path      = string<br/>      read_only = optional(bool, true)<br/>    }))<br/>    gcs = optional(object({<br/>      bucket        = string<br/>      read_only     = optional(bool, true)<br/>      mount_options = optional(list(string), [])<br/>    }))<br/>  }))</pre> | `[]` | no |

## Outputs

| Name | Description |
| ---- | ----------- |
| <a name="output_image_refs"></a> [image\_refs](#output\_image\_refs) | The signed image reference for each container, keyed by container name. Computed by ko/cosign before the Cloud Run Job is updated, so stable during apply. |
| <a name="output_job_etag"></a> [job\_etag](#output\_job\_etag) | The etag of the Cloud Run Job in each region, changes whenever the job definition changes. |
| <a name="output_job_ids"></a> [job\_ids](#output\_job\_ids) | The ID of the Cloud Run Job in each region. |
| <a name="output_job_name"></a> [job\_name](#output\_job\_name) | The name of the Cloud Run Job created in each region. |
<!-- END_TF_DOCS -->
