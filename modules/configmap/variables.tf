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
