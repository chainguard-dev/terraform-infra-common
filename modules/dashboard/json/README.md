# `dashboard/json`

This module modifies the dashboard_json to remove defaulted values that are known to generate no-op diffs.

```hcl
// Call module to generate cleaned up json
module "dashboard_json" {
  source = "chainguard-dev/common/infra//modules/dashboard/json"

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

// Create dashboard with cleaned up json.
resource "google_monitoring_dashboard" "dashboard" {
  dashboard_json = module.dashboard_json.json
}
```

The dashboard resource should now diff properly.

<!-- BEGIN_TF_DOCS -->
## Requirements

No requirements.

## Providers

No providers.

## Modules

No modules.

## Resources

No resources.

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_object"></a> [object](#input\_object) | Object to encode into JSON | `object` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_json"></a> [json](#output\_json) | n/a |
<!-- END_TF_DOCS -->
