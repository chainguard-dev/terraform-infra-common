# terraform-infra-common

[<img alt="Static Badge" src="https://img.shields.io/badge/terraform-%235835CC.svg?style=for-the-badge&logo=terraform&logoColor=white&link=https%3A%2F%2Fregistry.terraform.io%2Fmodules%2Fchainguard-dev%2Fcommon%2Finfra%2Flatest">](https://registry.terraform.io/modules/chainguard-dev/common/infra/latest)

A collection of common infrastructure modules that encapsulate Cloud Run and
GCP patterns.

## Modules

See [MODULES.md](./MODULES.md) for a summary of all available modules.

## Go helpers

The [httpmetrics package](./pkg/httpmetrics/README.md) provides HTTP instrumentation
and opt-in serving-revision metadata for HTTP and gRPC services.

## Usage

To use components in this library, you must provide the `project` in a
`provider.google` resource in your top-level main.tf:

```hcl
provider "google" {
  project = var.project
}
```

## Resource labels

Many modules add a `terraform-module` label and a label named after the module. Modules differ in which other labels they apply and whether they accept a `labels` map. Check the module's variables and resources before relying on a specific label.

When a module requires a `team` input, provide the owning team. Some modules also add a `squad` label with that value for compatibility. The `product` input and its default also vary by module.

## Submitting Changes

These modules are canonically located within a private Chainguard repository, and are continuously pushed from there to this repository.

If you would like to submit a PR, please do make one against this repository.
After the review process, someone at Chainguard will merge it into our internal repository,
and the change will then be pushed here.
