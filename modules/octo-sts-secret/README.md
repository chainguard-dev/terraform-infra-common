# `octo-sts-secret`

This module sets up a Cloud Secret with a GitHub token obtained from [octo-sts](https://github.com/octo-sts/app). This module is particularly useful when a native Google Cloud service requires a GitHub token to be provided in a Cloud Secret

The intended usage of this module:

```hcl
// Create a GitHub token rotated by octo-sts
module "gh-secret" {
  source  = "chainguard-dev/common/infra//modules/octo-sts-secret"

  name       = "my-secret"
  project_id = var.project_id
  region     = var.primary-region

  # The name of the GitHub org on which the secret will be scoped.
	github_org = "my-org"
  # The name of the GitHub repo on which the secret will be scoped.
	github_repo = "my-repo"
  # The name of the octo-sts policy.
	octosts_policy = "my-policy"

  # What the service accessing this secret will run as.
  service-account = google_service_account.foo.email

  # Optionally: channels to notify if this secret is manipulated.
  notification-channels = [ ... ]
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

| Name | Source | Version |
|------|--------|---------|
| <a name="module_gh-token-secret"></a> [gh-token-secret](#module\_gh-token-secret) | ../secret | n/a |
| <a name="module_this"></a> [this](#module\_this) | ../cron | n/a |

## Resources

| Name | Type |
|------|------|
| [google_secret_manager_secret_iam_binding.authorize-manage](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/secret_manager_secret_iam_binding) | resource |
| [google_service_account.octo-sts-rotator](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/service_account) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_github_org"></a> [github\_org](#input\_github\_org) | The GitHub organization for which the octo-sts token will be requested. | `string` | n/a | yes |
| <a name="input_github_repo"></a> [github\_repo](#input\_github\_repo) | The GitHub repository for which the octo-sts token will be requested. | `string` | n/a | yes |
| <a name="input_invokers"></a> [invokers](#input\_invokers) | List of user emails to grant invoker perimssions to invoke the job. | `list(string)` | `[]` | no |
| <a name="input_name"></a> [name](#input\_name) | The name to give the secret. | `string` | n/a | yes |
| <a name="input_notification_channels"></a> [notification\_channels](#input\_notification\_channels) | List of notification channels to alert. | `list(string)` | n/a | yes |
| <a name="input_octosts_policy"></a> [octosts\_policy](#input\_octosts\_policy) | The name of the octo-sts policy for which to request a token. | `string` | n/a | yes |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | n/a | `string` | n/a | yes |
| <a name="input_region"></a> [region](#input\_region) | The region to run the job. | `string` | `"us-east4"` | no |
| <a name="input_service_account"></a> [service\_account](#input\_service\_account) | The email of the service account that will access the secret. | `string` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_secret_id"></a> [secret\_id](#output\_secret\_id) | The ID of the secret. |
<!-- END_TF_DOCS -->
