# terraform-infra-common

A repository containing a collection of common infrastructure modules for
encapsulating common Cloud Run patterns.

## Usage

To use components in this library, you must provide the `project` in a
`provider.google` resource in your top-level main.tf:

```hcl
provider "google" {
  project = var.project
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
| <a name="module_cloudevent-broker"></a> [cloudevent-broker](#module\_cloudevent-broker) | ./modules/cloudevent-broker | n/a |
| <a name="module_layout"></a> [layout](#module\_layout) | ./modules/dashboard/sections/layout | n/a |
| <a name="module_logurl"></a> [logurl](#module\_logurl) | ./modules/dashboard/logurl | n/a |
| <a name="module_markdown"></a> [markdown](#module\_markdown) | ./modules/dashboard/sections/markdown | n/a |
| <a name="module_networking"></a> [networking](#module\_networking) | chainguard-dev/common/infra//modules/networking | n/a |
| <a name="module_recorder"></a> [recorder](#module\_recorder) | ./modules/cloudevent-recorder | n/a |
| <a name="module_width"></a> [width](#module\_width) | ./modules/dashboard/sections/width | n/a |

## Resources

| Name | Type |
|------|------|
| [google_monitoring_dashboard.dashboard](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/monitoring_dashboard) | resource |

## Inputs

No inputs.

## Outputs

No outputs.
<!-- END_TF_DOCS -->