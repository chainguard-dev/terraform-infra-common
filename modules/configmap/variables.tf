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

variable "require_squad" {
  description = "Whether to require squad variable to be specified"
  type        = bool
  default     = false
}

variable "squad" {
  description = "Squad label to apply to the secret."
  type        = string
  default     = ""

  validation {
    condition     = !var.require_squad || var.squad != ""
    error_message = "squad must be specified or disable check by setting require_squad = false"
  }
}
