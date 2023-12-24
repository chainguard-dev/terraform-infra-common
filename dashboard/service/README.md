# `dashboard/service`

This module provisions a Google Cloud Monitoring dashboard for a regionalized Cloud Run service.

It assumes the service has the same name in all regions.

```hcl
// Create a network with several regional subnets
module "networking" {
  source = "chainguard-dev/glue/cloudrun//networking"

  name       = "my-networking"
  project_id = var.project_id
  regions    = [...]
}

// Run a regionalized cloud run service "frontend" to serve requests.
module "frontend" {
  source = "chainguard-dev/glue/cloudrun//regional-go-service"

  project_id = var.project_id
  name       = "frontend"
  regions    = module.networking.regional-networks

  service_account = google_service_account.frontend.email
  containers = {
    "frontend" = {
      source = {
        working_dir = path.module
        importpath  = "./cmd/frontend"
      }
      ports = [{ container_port = 8080 }]
    }
  }
}

// Set up a dashboard for a regionalized service named "frontend".
module "service-dashboard" {
  source       = "chainguard-dev/glue/cloudrun//dashboard/service"
  service_name = "frontend"
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
| <a name="module_alerts"></a> [alerts](#module\_alerts) | ../sections/alerts | n/a |
| <a name="module_grpc"></a> [grpc](#module\_grpc) | ../sections/grpc | n/a |
| <a name="module_http"></a> [http](#module\_http) | ../sections/http | n/a |
| <a name="module_layout"></a> [layout](#module\_layout) | ../sections/layout | n/a |
| <a name="module_logs"></a> [logs](#module\_logs) | ../sections/logs | n/a |
| <a name="module_resources"></a> [resources](#module\_resources) | ../sections/resources | n/a |
| <a name="module_width"></a> [width](#module\_width) | ../sections/width | n/a |

## Resources

| Name | Type |
|------|------|
| [google_monitoring_dashboard.dashboard](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/monitoring_dashboard) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_alerts"></a> [alerts](#input\_alerts) | Alerting policies to add to the dashboard. | `list(string)` | `[]` | no |
| <a name="input_grpc_service_name"></a> [grpc\_service\_name](#input\_grpc\_service\_name) | Name of the GRPC service(s) to monitor | `string` | `""` | no |
| <a name="input_labels"></a> [labels](#input\_labels) | Additional labels to apply to the dashboard. | `map` | `{}` | no |
| <a name="input_service_name"></a> [service\_name](#input\_service\_name) | Name of the service(s) to monitor | `string` | n/a | yes |

## Outputs

No outputs.
<!-- END_TF_DOCS -->
