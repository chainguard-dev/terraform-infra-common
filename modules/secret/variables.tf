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

variable "version_managers" {
  description = "Member-style identities (e.g. group:oncall@my-corp.dev) that manage this secret's versions: add, list, get, enable, disable, and destroy them, and view the secret. They can't read version payloads, and the abnormal-access alert ignores their version management. Only user:, group:, and serviceAccount: members are accepted, because the alert recognizes identities by email. The module owns the secret's roles/secretmanager.secretVersionManager and roles/secretmanager.viewer bindings once this is set, so don't grant those roles on the secret elsewhere."
  type        = list(string)
  default     = []
  nullable    = false

  validation {
    condition     = alltrue([for m in var.version_managers : can(regex("^(user|group|serviceAccount):[^\\s\"\\\\:@]+@[^\\s\"\\\\:@]+$", m))])
    error_message = "version_managers entries must be user:, group:, or serviceAccount: followed by an email (e.g. group:oncall@my-corp.dev)."
  }
}

variable "group_members" {
  description = "Member emails of each group in authorized-adder, authorized-adders, and version_managers, keyed by the group's email. Audit logs name the member who acted, never the group, so the abnormal-access alert recognizes a group's members only through this map. Resolve it from the groups (e.g. with the group-members module) so it follows membership changes. Grants no IAM."
  type        = map(list(string))
  default     = {}
  nullable    = false

  validation {
    condition = alltrue([
      for group in keys(var.group_members) : contains([
        for m in concat([var.authorized-adder], var.authorized-adders, var.version_managers) :
        lower(trimprefix(m, "group:")) if startswith(m, "group:")
      ], lower(group))
    ])
    error_message = "Each group_members key must be the email of a group: member of authorized-adder, authorized-adders, or version_managers, without the group: prefix."
  }

  validation {
    condition     = alltrue([for e in flatten(values(var.group_members)) : can(regex("^[^\\s\"\\\\:@]+@[^\\s\"\\\\:@]+$", e))])
    error_message = "group_members values must be bare emails, without a user: or serviceAccount: prefix."
  }
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

variable "alert_documentation" {
  description = "Markdown documentation attached to the abnormal-access alert (e.g. what to check and a Logs Explorer link). Empty (default) attaches none."
  type        = string
  default     = ""
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
  description = "Unused: cost attribution comes from Resource Manager tags, not labels. Kept so existing callers still plan."
  type        = string
  default     = null
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
