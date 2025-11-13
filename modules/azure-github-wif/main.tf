resource "azuread_application" "this" {
  display_name = var.azure_app_name
}

resource "azuread_service_principal" "this" {
  client_id = azuread_application.this.client_id
}

resource "azuread_application_federated_identity_credential" "this" {
  application_id = "/applications/${azuread_application.this.object_id}"
  display_name   = var.azure_app_name
  description    = var.description
  issuer         = "https://token.actions.githubusercontent.com"
  subject        = var.subject
  audiences      = ["api://AzureADTokenExchange"]
}

resource "azurerm_role_assignment" "this_access_subscription_role" {
  scope                = var.subscription_id
  role_definition_name = "Reader"
  principal_id         = azuread_service_principal.this.object_id
}

resource "azurerm_role_assignment" "this_resource_group_vms_role" {
  scope                = var.resource_group_id
  role_definition_name = "Contributor"
  principal_id         = azuread_service_principal.this.object_id
}
