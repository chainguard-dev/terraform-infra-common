<!-- BEGIN_TF_DOCS -->
## Requirements

No requirements.

## Providers

No providers.

## Modules

| Name | Source | Version |
|------|--------|---------|
| <a name="module_github_ci_processor"></a> [github\_ci\_processor](#module\_github\_ci\_processor) | ../ | n/a |
| <a name="module_github_issue_processor"></a> [github\_issue\_processor](#module\_github\_issue\_processor) | ../ | n/a |
| <a name="module_github_merged_pr_processor"></a> [github\_merged\_pr\_processor](#module\_github\_merged\_pr\_processor) | ../ | n/a |
| <a name="module_github_pr_processor"></a> [github\_pr\_processor](#module\_github\_pr\_processor) | ../ | n/a |
| <a name="module_github_specific_repo_processor"></a> [github\_specific\_repo\_processor](#module\_github\_specific\_repo\_processor) | ../ | n/a |

## Resources

No resources.

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_broker"></a> [broker](#input\_broker) | A map from each of the input region names to the name of the Broker topic in that region | `map(string)` | n/a | yes |
| <a name="input_notification_channels"></a> [notification\_channels](#input\_notification\_channels) | Notification channels for alerts | `list(string)` | n/a | yes |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | GCP Project ID | `string` | n/a | yes |
| <a name="input_regions"></a> [regions](#input\_regions) | Regions to deploy in | <pre>map(object({<br/>    network = string<br/>    subnet  = string<br/>  }))</pre> | n/a | yes |
| <a name="input_team"></a> [team](#input\_team) | Team label to apply to resources | `string` | n/a | yes |
| <a name="input_workqueue_dispatcher_name"></a> [workqueue\_dispatcher\_name](#input\_workqueue\_dispatcher\_name) | Name of the workqueue dispatcher service | `string` | n/a | yes |

## Outputs

No outputs.
<!-- END_TF_DOCS -->
