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
