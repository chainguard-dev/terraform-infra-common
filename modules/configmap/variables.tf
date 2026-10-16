variable "project_id" {
  type = string
}

variable "name" {
  description = "The name to give the secret."
  type        = string
}

variable "data" {
  description = "The data to place in the secret."
  type        = string
}

variable "service-account" {
  description = "The email of the service account that will access the secret."
  type        = string
}

variable "notification-channels" {
  description = "The channels to notify if the configuration data is improperly accessed."
  type        = list(string)
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

variable "project_number" {
  description = <<EOD
Optional project number of project_id, used to build tag binding parents. When
set, the module does not read the google_project data source. Pass it when a
caller uses a module-level depends_on, which would otherwise defer the data
source to apply and force every tag binding to be replaced. Must be known at
plan time (for example the number of an existing project read from a
`google_project` data source with no `depends_on`); leave it unset when the
project is created in the same apply.
EOD
  type        = string
  default     = null

  validation {
    condition     = var.project_number == null || can(regex("^[0-9]+$", var.project_number))
    error_message = "project_number must contain only digits."
  }
}

variable "resource_manager_tags" {
  description = "Resource Manager tags to bind to the backing secret, as tagKeys/<id> => tagValues/<id>."
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
