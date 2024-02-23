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
| <a name="provider_google"></a> [google](#provider\_google) | 5.10.0 |

## Modules

| Name | Source | Version |
|------|--------|---------|
| <a name="module_bucket-events"></a> [bucket-events](#module\_bucket-events) | ./modules/bucket-events | n/a |
| <a name="module_cloudevent-broker"></a> [cloudevent-broker](#module\_cloudevent-broker) | chainguard-dev/common/infra//modules/cloudevent-broker | n/a |
| <a name="module_networking"></a> [networking](#module\_networking) | chainguard-dev/common/infra//modules/networking | n/a |

## Resources

| Name | Type |
|------|------|
| [google_storage_bucket.bucket](https://registry.terraform.io/providers/hashicorp/google/latest/docs/data-sources/storage_bucket) | data source |

## Inputs

No inputs.

## Outputs

No outputs.
<!-- END_TF_DOCS -->