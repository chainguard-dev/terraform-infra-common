variable "name" {
  type = string
}

variable "project" {
  type = string
}

variable "network" {
  description = "The network to deploy the cluster in."
  type        = string
}

variable "region" {
  description = "Always create a regional cluster since GKE doesn't charge differently for regional/zonal clusters. Rather, we configure the node locations using `var.zones`"
  type        = string
}

variable "squad" {
  description = "squad label to apply to the service."
  type        = string
  default     = "unknown"
}

variable "zones" {
  default     = null
  description = "If specified, will spread nodes across these zones"
  type        = list(string)
}

variable "subnetwork" {
  description = "The subnetwork to deploy the cluster in."
  type        = string
}

variable "extra_roles" {
  type        = map(string)
  default     = {}
  description = "Extra roles to add to the cluster's default service account"
}

variable "release_channel" {
  type        = string
  default     = "REGULAR"
  description = "GKE release channel"
}

variable "deletion_protection" {
  type        = bool
  default     = true
  description = "Toggle to prevent accidental deletion of resources."
}

variable "enable_private_nodes" {
  type        = bool
  default     = false
  description = "Enable private nodes by default"
}

variable "product" {
  description = "Product label to apply to the service."
  type        = string
  default     = "unknown"
}
