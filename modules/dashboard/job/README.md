# `dashboard/job`

This module provisions a Google Cloud Monitoring dashboard for a Cloud Run job.

It assumes the service has the same name in all regions.

```hcl
// Run a cloud run job named "sync" to perform some work.
resource "google_cloud_run_v2_job" "sync" {
  name     = "sync"

  //...
  template {
    //...
    containers {
      image = "..."
    }
  }
}

// Set up a dashboard for a regionalized job named "sync".
module "job-dashboard" {
  source       = "chainguard-dev/common/infra//modules/dashboard/job"
  service_name = google_cloud_run_v2_job.name
}
```

The dashboard it creates includes widgets for job logs, CPU and memory
utilization, startup latency, and sent/received bytes.

<!-- BEGIN_TF_DOCS -->
## Requirements

No requirements.

## Providers

No providers.

## Modules

| Name | Source | Version |
|------|--------|---------|
| <a name="module_dashboard"></a> [dashboard](#module\_dashboard) | ../ | n/a |
| <a name="module_errgrp"></a> [errgrp](#module\_errgrp) | ../sections/errgrp | n/a |
| <a name="module_layout"></a> [layout](#module\_layout) | ../sections/layout | n/a |
| <a name="module_logs"></a> [logs](#module\_logs) | ../sections/logs | n/a |
| <a name="module_resources"></a> [resources](#module\_resources) | ../sections/resources | n/a |
| <a name="module_width"></a> [width](#module\_width) | ../sections/width | n/a |

## Resources

No resources.

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_job_name"></a> [job\_name](#input\_job\_name) | Name of the job(s) to monitor | `string` | n/a | yes |
| <a name="input_labels"></a> [labels](#input\_labels) | Additional labels to apply to the dashboard. | `map` | `{}` | no |
| <a name="input_notification_channels"></a> [notification\_channels](#input\_notification\_channels) | List of notification channels to alert. | `list(string)` | n/a | yes |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | ID of the GCP project | `string` | n/a | yes |

## Outputs

No outputs.
<!-- END_TF_DOCS -->
