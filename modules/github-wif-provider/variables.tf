variable "project_id" {
  type = string
}

variable "name" {
  description = "The name to give the provider pool."
  type        = string
}

variable "notification_channels" {
  description = "The list of notification channels to alert when this policy fires."
  type        = list(string)
}

variable "github_org" {
  description = "The GitHub organizantion to grant access to. Eg: 'chainguard-dev'."
  type        = string
}

variable "github_org_id" {
  description = "Numeric ID of github_org (for example '123456789'). When set, the provider also requires assertion.repository_owner_id to match, so a different organization that later takes the github_org name is rejected. Empty skips the check."
  type        = string
  default     = ""
  validation {
    condition     = can(regex("^[0-9]*$", var.github_org_id))
    error_message = "github_org_id must be the numeric GitHub organization ID, or empty."
  }
}

variable "allowed_repositories" {
  description = "Repositories (\"<github_org>/<repo>\") whose tokens this pool accepts. List every repository that binds a service account in this pool. Empty keeps the org-wide condition."
  type        = list(string)
  default     = []
  validation {
    condition     = alltrue([for r in var.allowed_repositories : can(regex("^[A-Za-z0-9-]+/[A-Za-z0-9._-]+$", r)) && startswith(r, "${var.github_org}/")])
    error_message = "Each allowed_repositories entry must be \"<github_org>/<repo>\"; the provider already rejects other orgs."
  }
}
