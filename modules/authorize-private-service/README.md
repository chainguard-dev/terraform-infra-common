# `authorize-private-service`

This module takes a reference to a private Cloud Run service, authorizes the
named service account to invoke that service, and returns its URLs.

Effectively this module encapsulates:
```
{project, region, name}  --(authorize)-->  {uri, deterministic_uri}
```

`uri` is the hashed `*.a.run.app` URL. `deterministic_uri` is
`https://<name>-<project number>.<region>.run.app`, which the service can
compute before it exists. A caller that mints its ID token for the URL it dials
should dial `deterministic_uri` when the service checks the token's audience
against its own URL.

<!-- BEGIN_TF_DOCS -->
## Requirements

| Name | Version |
| ---- | ------- |
| <a name="requirement_terraform"></a> [terraform](#requirement\_terraform) | >= 1.9 |
| <a name="requirement_google"></a> [google](#requirement\_google) | >= 8.7.0 |

## Providers

| Name | Version |
| ---- | ------- |
| <a name="provider_google"></a> [google](#provider\_google) | >= 8.7.0 |

## Modules

No modules.

## Resources

| Name | Type |
| ---- | ---- |
| [google_cloud_run_v2_service_iam_member.authorize-calls](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/cloud_run_v2_service_iam_member) | resource |
| [google_cloud_run_v2_service.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/data-sources/cloud_run_v2_service) | data source |

## Inputs

| Name | Description | Type | Default | Required |
| ---- | ----------- | ---- | ------- | :------: |
| <a name="input_name"></a> [name](#input\_name) | The name of the Cloud Run service in this region. | `string` | n/a | yes |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | n/a | `string` | n/a | yes |
| <a name="input_region"></a> [region](#input\_region) | The region in which this Cloud Run service is based. | `string` | n/a | yes |
| <a name="input_service-account"></a> [service-account](#input\_service-account) | The email of the service account being authorized to invoke the private Cloud Run service. | `string` | n/a | yes |

## Outputs

| Name | Description |
| ---- | ----------- |
| <a name="output_deterministic_uri"></a> [deterministic\_uri](#output\_deterministic\_uri) | The private Cloud Run service's deterministic URL, https://<name>-<project number>.<region>.run.app, or null if Cloud Run reports none. A caller that dials it presents an ID token for that audience, which the service can know before it exists. |
| <a name="output_invocation_grant"></a> [invocation\_grant](#output\_invocation\_grant) | Service invocation grant managed by this module. Excludes inherited and out-of-band IAM grants. |
| <a name="output_uri"></a> [uri](#output\_uri) | The URI of the private Cloud Run service. |
<!-- END_TF_DOCS -->
