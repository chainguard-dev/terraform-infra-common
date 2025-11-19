# `github-wif-provider`

This module sets up a Google Workload Identity Pool and Provider that are set up
to project the GitHub token's claims into attributes in a way that the
[`github-gsa`](../github-gsa/README.md) can match to authorize access.  For
examples of how these are used together see the `github-gsa` README.

<!-- BEGIN_TF_DOCS -->
## Requirements

No requirements.

## Providers

| Name | Version |
|------|---------|
| <a name="provider_google-beta"></a> [google-beta](#provider\_google-beta) | n/a |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [google-beta_google_iam_workload_identity_pool.this](https://registry.terraform.io/providers/hashicorp/google-beta/latest/docs/resources/google_iam_workload_identity_pool) | resource |
| [google-beta_google_iam_workload_identity_pool_provider.this](https://registry.terraform.io/providers/hashicorp/google-beta/latest/docs/resources/google_iam_workload_identity_pool_provider) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_github_org"></a> [github\_org](#input\_github\_org) | The GitHub organizantion to grant access to. Eg: 'chainguard-dev'. | `string` | n/a | yes |
| <a name="input_name"></a> [name](#input\_name) | The name to give the provider pool. | `string` | n/a | yes |
| <a name="input_notification_channels"></a> [notification\_channels](#input\_notification\_channels) | The list of notification channels to alert when this policy fires. | `list(string)` | n/a | yes |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | n/a | `string` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_pool_name"></a> [pool\_name](#output\_pool\_name) | n/a |
<!-- END_TF_DOCS -->
