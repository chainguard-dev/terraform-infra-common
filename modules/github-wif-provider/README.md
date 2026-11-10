# `github-wif-provider`

This module sets up a Google Workload Identity Pool and Provider that are set up
to project the GitHub token's claims into attributes in a way that the
[`github-gsa`](../github-gsa/README.md) can match to authorize access.  For
examples of how these are used together see the `github-gsa` README.

By default the provider accepts a token from any repository in `github_org`.
Set `allowed_repositories` to every repository that binds a service account in
the pool, so tokens from other repositories in the org cannot exchange into it.
Setting `github_org_id` also pins the numeric organization ID. Adding either to
a live pool takes effect for every token exchange as soon as it is applied, so
derive the list from the pool's STS `ExchangeToken` audit logs first.

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
| [google_iam_workload_identity_pool.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/iam_workload_identity_pool) | resource |
| [google_iam_workload_identity_pool_provider.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/iam_workload_identity_pool_provider) | resource |

## Inputs

| Name | Description | Type | Default | Required |
| ---- | ----------- | ---- | ------- | :------: |
| <a name="input_allowed_repositories"></a> [allowed\_repositories](#input\_allowed\_repositories) | Repositories ("<github\_org>/<repo>") whose tokens this pool accepts. List every repository that binds a service account in this pool. Empty keeps the org-wide condition. | `list(string)` | `[]` | no |
| <a name="input_github_org"></a> [github\_org](#input\_github\_org) | The GitHub organizantion to grant access to. Eg: 'chainguard-dev'. | `string` | n/a | yes |
| <a name="input_github_org_id"></a> [github\_org\_id](#input\_github\_org\_id) | Numeric ID of github\_org (for example '123456789'). When set, the provider also requires assertion.repository\_owner\_id to match, so a different organization that later takes the github\_org name is rejected. Empty skips the check. | `string` | `""` | no |
| <a name="input_name"></a> [name](#input\_name) | The name to give the provider pool. | `string` | n/a | yes |
| <a name="input_notification_channels"></a> [notification\_channels](#input\_notification\_channels) | The list of notification channels to alert when this policy fires. | `list(string)` | n/a | yes |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | n/a | `string` | n/a | yes |

## Outputs

| Name | Description |
| ---- | ----------- |
| <a name="output_pool_name"></a> [pool\_name](#output\_pool\_name) | n/a |
<!-- END_TF_DOCS -->
