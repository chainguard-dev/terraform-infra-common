# `observability-role`

This module creates a per-project custom IAM role carrying the union of the
permissions of `roles/monitoring.metricWriter`, `roles/cloudtrace.agent`, and
`roles/cloudprofiler.agent`.

The service modules (`regional-go-service`, `regional-service`, `cron`,
`regional-go-cron`, and their wrappers) grant those three built-in roles to
each service's service account at the project level, costing three entries in
the project IAM policy per service. GCP caps an IAM policy at 1,500 member
entries, so projects with hundreds of services approach the limit quickly.
Creating this role once per project and passing it to the service modules via
`observability_role` collapses the three entries into one.

```hcl
// Once per project, in a stack that applies before the services.
module "observability-role" {
  source = "chainguard-dev/common/infra//modules/observability-role"

  project_id = var.project_id
}

module "foo-service" {
  source     = "chainguard-dev/common/infra//modules/regional-go-service"
  project_id = var.project_id
  name       = "foo"

  // One project IAM policy entry instead of three.
  observability_role = module.observability-role.id

  ...
}
```

Services in stacks that cannot reference the module instance directly can
construct the id by convention: `projects/${var.project_id}/roles/serviceObservability`.

Note: the `id` output is deliberately assembled from configuration attributes
(`project`, `role_id`) rather than the resource's computed `id`, so that it is
known at plan time — the service modules use `observability_role` in `count`
expressions, which cannot depend on values only known after apply. Don't
"simplify" it to the computed attribute.

<!-- BEGIN_TF_DOCS -->
## Requirements

No requirements.

## Providers

| Name | Version |
| ---- | ------- |
| <a name="provider_google"></a> [google](#provider\_google) | n/a |

## Modules

No modules.

## Resources

| Name | Type |
| ---- | ---- |
| [google_project_iam_custom_role.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/project_iam_custom_role) | resource |

## Inputs

| Name | Description | Type | Default | Required |
| ---- | ----------- | ---- | ------- | :------: |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | The project in which to create the custom role. | `string` | n/a | yes |
| <a name="input_role_id"></a> [role\_id](#input\_role\_id) | The role\_id of the custom role. | `string` | `"serviceObservability"` | no |
| <a name="input_title"></a> [title](#input\_title) | Human-readable title of the custom role. | `string` | `"Service Observability"` | no |

## Outputs

| Name | Description |
| ---- | ----------- |
| <a name="output_id"></a> [id](#output\_id) | Fully-qualified role id (projects/{project}/roles/{role\_id}), for use as the observability\_role input of the service modules. |
<!-- END_TF_DOCS -->
