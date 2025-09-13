<!-- BEGIN_TF_DOCS -->
## Requirements

No requirements.

## Providers

| Name | Version |
|------|---------|
| <a name="provider_azuread"></a> [azuread](#provider\_azuread) | n/a |
| <a name="provider_azurerm"></a> [azurerm](#provider\_azurerm) | n/a |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [azuread_application.this](https://registry.terraform.io/providers/hashicorp/azuread/latest/docs/resources/application) | resource |
| [azuread_application_federated_identity_credential.this](https://registry.terraform.io/providers/hashicorp/azuread/latest/docs/resources/application_federated_identity_credential) | resource |
| [azuread_service_principal.this](https://registry.terraform.io/providers/hashicorp/azuread/latest/docs/resources/service_principal) | resource |
| [azurerm_role_assignment.this_access_subscription_role](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/role_assignment) | resource |
| [azurerm_role_assignment.this_resource_group_vms_role](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/role_assignment) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_azure_app_name"></a> [azure\_app\_name](#input\_azure\_app\_name) | The name to give the Azure AD Application. | `string` | n/a | yes |
| <a name="input_description"></a> [description](#input\_description) | The description to give the Azure AD Application. | `string` | `"OIDC for GitHub Actions"` | no |
| <a name="input_resource_group_id"></a> [resource\_group\_id](#input\_resource\_group\_id) | The resource group ID to give permissions to use for the Azure AD Application. | `string` | n/a | yes |
| <a name="input_subject"></a> [subject](#input\_subject) | The subject to use for the Azure AD Application. Should be in the format 'repo:<org>/<repo>:ref:refs/heads/<branch>' or 'repo:,<org>/<repo>:pull\_request' | `string` | n/a | yes |
| <a name="input_subscription_id"></a> [subscription\_id](#input\_subscription\_id) | The subscription ID to give permissions to use for the Azure AD Application. | `string` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_azure_this_client_id"></a> [azure\_this\_client\_id](#output\_azure\_this\_client\_id) | n/a |
<!-- END_TF_DOCS -->
