# Dashboards

The modules in this directory define [`google_monitoring_dashboard`](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/monitoring_dashboard) resources in a repeatable structured way.

This module itself creates a dashboard that cleans up defaulted no-op changes.

- The [Service](service/README.md) and [Job](job/README.md) modules define pre-configured dashboards for Cloud Run services and Cloud Run jobs, respectively.
- The [`cloudevent-receiver`](cloudevent-receiver/README.md) module defines a pre-configured dashboard for a Cloud Run-based event handler receiving events from a `cloudevent-trigger`.
- The modules in [`./widgets`](widgets/) define the widgets used by the dashboards, in a way that can be reused to create custom dashboards.

```hcl
// Call module to generate cleaned up json
module "dashboard" {
  source = "chainguard-dev/common/infra//modules/dashboard"

  object = {
    displayName      = "Elastic Build"
    labels           = { "elastic-builds" : "" }
    dashboardFilters = []

    mosaicLayout = {
      columns = module.width.size
      tiles   = module.layout.tiles,
    }
  }
}
```

The dashboard resource should now diff properly.

<!-- BEGIN_TF_DOCS -->
## Requirements

No requirements.

## Providers

| Name | Version |
|------|---------|
| <a name="provider_google"></a> [google](#provider\_google) | n/a |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [google_monitoring_dashboard.dashboard](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/monitoring_dashboard) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_object"></a> [object](#input\_object) | Object to encode into JSON | `any` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_json"></a> [json](#output\_json) | n/a |
<!-- END_TF_DOCS -->
