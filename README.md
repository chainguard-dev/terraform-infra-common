# terraform-infra-common

[<img alt="Static Badge" src="https://img.shields.io/badge/terraform-%235835CC.svg?style=for-the-badge&logo=terraform&logoColor=white&link=https%3A%2F%2Fregistry.terraform.io%2Fmodules%2Fchainguard-dev%2Fcommon%2Finfra%2Flatest">](https://registry.terraform.io/modules/chainguard-dev/common/infra/latest)



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

## Resource Labeling Convention

All modules in this repository follow a consistent labeling pattern for GCP cost allocation and resource organization:

```hcl
locals {
  default_labels = {
    basename(abspath(path.module)) = var.name
    terraform-module               = basename(abspath(path.module))
  }

  squad_label = var.squad != "" ? {
    squad = var.squad
    team  = var.squad
  } : {}

  merged_labels = merge(local.default_labels, local.squad_label, var.labels)
}
```

This pattern:
- **Enables cost tracking** to break down each module by use
- **Maintains consistency** across all infrastructure modules
- **Supports team attribution** through squad/team labels
- **Allows custom labels** via the `labels` variable
- **Provides module identification** via the `terraform-module` label

The `basename(abspath(path.module))` automatically derives the module name (e.g., "gke", "redis", "workqueue") without requiring hardcoded values.
