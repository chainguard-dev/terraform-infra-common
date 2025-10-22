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



variable "team" {
  description = "Team label to apply to resources (replaces deprecated 'squad')."
  type        = string
  default     = ""
}

variable "product" {
  description = "Product label to apply to the service."
  type        = string
  default     = "unknown"
}
