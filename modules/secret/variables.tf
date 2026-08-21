variable "project_id" {
  type = string
}

variable "name" {
  description = "The name to give the secret."
  type        = string
}

variable "authorized-adder" {
  description = "A member-style representation of the identity authorized to add new secret values (e.g. group:oncall@my-corp.dev)."
  type        = string
}

variable "authorized-adders" {
  description = "List of additional identities authorized to add new secret values (e.g. [\"serviceAccount:sa@project.iam.gserviceaccount.com\"]). Use this when multiple identities need version adder permissions."
  type        = list(string)
  default     = []
}

variable "service-account" {
  description = "(Deprecated: Use service-accounts instead) The email of the service account that will access the secret."
  type        = string
  default     = ""
}

variable "service-accounts" {
  description = "The emails of the service accounts that will access the secret."
  type        = list(string)
  default     = []

  validation {
    # To support the legacy service-account variable, ensure that either that var is
    # non-empty, or service-accounts is non-empty.
    condition     = var.service-account != "" || length(var.service-accounts) > 0
    error_message = "Must provide at least one value in service-accounts"
  }
}

variable "notification-channels" {
  description = "The channels to notify if the configuration data is improperly accessed."
  type        = list(string)
}

variable "create_placeholder_version" {
  description = "Whether to create a placeholder secret version to avoid bad reference on first deploy."
  type        = bool
  default     = false
}

variable "labels" {
  description = "Labels to apply to the secret."
  type        = map(string)
  default     = {}
}

variable "replication_locations" {
  description = "List of GCP regions for user_managed replication. When null (default), uses automatic replication."
  type        = list(string)
  default     = null
}

variable "team" {
  description = "Team label to apply to resources (replaces deprecated 'squad')."
  type        = string
}

variable "product" {
  description = "Product label to apply to the service."
  type        = string
  default     = "unknown"
}

variable "resource_manager_tags" {
  description = "Resource Manager tags to bind to the secret, as tagKeys/<id> => tagValues/<id>."
  type        = map(string)
  default     = {}
  nullable    = false

  validation {
    condition = alltrue([
      for key, value in var.resource_manager_tags :
      can(regex("^tagKeys/[0-9]+$", key)) && can(regex("^tagValues/[0-9]+$", value))
    ])
    error_message = "resource_manager_tags keys must be tagKeys/<numeric-id> and values must be tagValues/<numeric-id>."
  }
}
