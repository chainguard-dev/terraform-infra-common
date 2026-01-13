# `ocistatus`

This module encapsulates the creation of a Google Artifact Registry repository
for storing OCI statusmanager attestations. It provisions the repository with
appropriate cleanup policies and grants write access to the specified service
account.

```hcl
module "layer_attestations" {
  source = "chainguard-dev/common/infra//modules/ocistatus"

  project_id      = var.project_id
  name            = "${var.name}-layer"
  location        = var.primary_region
  service_account = google_service_account.foo.member
}

module "layer_reconciler" {
  source = "chainguard-dev/common/infra//modules/regional-go-reconciler"

  # ... other config ...

  containers = {
    reconciler = {
      # ... other config ...
      env = [{
        name  = "STATUSMANAGER_REPOSITORY"
        value = module.layer_attestations.attestations_path
      }]
    }
  }
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

No modules.

## Resources

| Name | Type |
|------|------|
| [google_artifact_registry_repository.attestations](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/artifact_registry_repository) | resource |
| [google_artifact_registry_repository_iam_member.writer](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/artifact_registry_repository_iam_member) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_cleanup_policy_older_than"></a> [cleanup\_policy\_older\_than](#input\_cleanup\_policy\_older\_than) | Duration after which untagged images are deleted (e.g. 86400s for 1 day). | `string` | `"86400s"` | no |
| <a name="input_location"></a> [location](#input\_location) | The location (region) for the Artifact Registry repository. | `string` | n/a | yes |
| <a name="input_name"></a> [name](#input\_name) | The name for the Artifact Registry repository. | `string` | n/a | yes |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | The GCP project ID where the repository will be created. | `string` | n/a | yes |
| <a name="input_service_account"></a> [service\_account](#input\_service\_account) | The service account member (e.g. serviceAccount:foo@project.iam.gserviceaccount.com) to grant write access. | `string` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_attestations_path"></a> [attestations\_path](#output\_attestations\_path) | The full path for storing attestations (for use with STATUSMANAGER\_REPOSITORY). |
| <a name="output_registry_uri"></a> [registry\_uri](#output\_registry\_uri) | The registry URI of the Artifact Registry repository. |
| <a name="output_repository_id"></a> [repository\_id](#output\_repository\_id) | The ID of the Artifact Registry repository. |
<!-- END_TF_DOCS -->
