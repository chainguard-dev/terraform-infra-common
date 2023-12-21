# `dashboard/cloudevent-receiver`

This module provisions a Google Cloud Monitoring dashboard for a regionalized Cloud Run service that receives Cloud Events from one or more `cloudevent-trigger`.

It assumes the service has the same name in all regions.

```hcl
// Create a network with several regional subnets
module "networking" {
  source = "chainguard-dev/glue/cloudrun//networking"

  name       = "my-networking"
  project_id = var.project_id
  regions    = [...]
}

// Run a regionalized cloud run service "receiver" to handle events.
resource "google_cloud_run_v2_service" "receiver" {
  for_each = module.networking.regional-networks
  name     = "receiver"

  //...
  template {
    //...
    containers {
      image = "..."
    }
  }
}

module "cloudevent-trigger" {
  for_each = module.networking.regional-networks

  source = "chainguard-dev/glue/cloudrun//cloudevent-trigger"

  name       = "my-trigger"
  project_id = var.project_id
  broker     = module.cloudevent-broker.broker[each.key]
  filter     = { "type" : "dev.chainguard.foo" }

  depends_on = [google_cloud_run_v2_service.sockeye]
  private-service = {
    region = each.key
    name   = google_cloud_run_v2_service.receiver[each.key].name
  }
}

// Set up a dashboard for a regionalized event handler named "receiver".
module "receiver-dashboard" {
  source       = "chainguard-dev/glue/cloudrun//dashboard/cloudevent-receiver"
  service_name = "receiver"

  triggers = {
    "type dev.chainguard.foo": "my-trigger"
  }
}
```

The dashboard it creates includes widgets for service logs, request count, latency (p50,p95,p99), instance count grouped by revision, CPU and memory utilization, startup latency, and sent/received bytes.

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
| <a name="module_http"></a> [http](#module\_http) | ../sections/http | n/a |
| <a name="module_layout"></a> [layout](#module\_layout) | ../sections/layout | n/a |
| <a name="module_logs"></a> [logs](#module\_logs) | ../sections/logs | n/a |
| <a name="module_resources"></a> [resources](#module\_resources) | ../sections/resources | n/a |
| <a name="module_subscription"></a> [subscription](#module\_subscription) | ../sections/subscription | n/a |
| <a name="module_width"></a> [width](#module\_width) | ../sections/width | n/a |

## Resources

| Name | Type |
|------|------|
| [google_monitoring_dashboard.dashboard](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/monitoring_dashboard) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_labels"></a> [labels](#input\_labels) | Additional labels to apply to the dashboard. | `map` | `{}` | no |
| <a name="input_service_name"></a> [service\_name](#input\_service\_name) | Name of the service(s) to monitor | `string` | n/a | yes |
| <a name="input_triggers"></a> [triggers](#input\_triggers) | A mapping from a descriptive name to a subscription name prefix. | `map(string)` | n/a | yes |

## Outputs

No outputs.
<!-- END_TF_DOCS -->
