variable "azure_app_name" {
  description = "The name to give the Azure AD Application."
  type        = string
}

variable "description" {
  description = "The description to give the Azure AD Application."
  default     = "OIDC for GitHub Actions"
  type        = string
}

variable "subject" {
  description = "The subject to use for the Azure AD Application. Should be in the format 'repo:<org>/<repo>:ref:refs/heads/<branch>' or 'repo:,<org>/<repo>:pull_request'"
  type        = string
}

variable "resource_group_id" {
  description = "The resource group ID to give permissions to use for the Azure AD Application."
  type        = string
}

variable "subscription_id" {
  description = "The subscription ID to give permissions to use for the Azure AD Application."
  type        = string
}
