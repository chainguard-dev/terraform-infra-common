# `authorize-private-service`

This module takes a reference to a private Cloud Run service, authorizes the
named service account to invoke that service, and returns the URI.

Effectively this module encapsulates:
```
{project, region, name}  --(authorize)-->  {uri}
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
| [google_cloud_run_v2_service_iam_member.authorize-calls](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/cloud_run_v2_service_iam_member) | resource |
| [google_cloud_run_v2_service.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/data-sources/cloud_run_v2_service) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_name"></a> [name](#input\_name) | The name of the Cloud Run service in this region. | `string` | n/a | yes |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | n/a | `string` | n/a | yes |
| <a name="input_region"></a> [region](#input\_region) | The region in which this Cloud Run service is based. | `string` | n/a | yes |
| <a name="input_service-account"></a> [service-account](#input\_service-account) | The email of the service account being authorized to invoke the private Cloud Run service. | `string` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_uri"></a> [uri](#output\_uri) | The URI of the private Cloud Run service. |
<!-- END_TF_DOCS -->