variable "name" {}

variable "project" {}

variable "network" {}

variable "region" {
  description = "Always create a regional cluster since GKE doesn't charge differently for regional/zonal clusters. Rather, we configure the node locations using `var.zones`"
}

variable "require_squad" {
  description = "Whether to require squad variable to be specified"
  type        = bool
  default     = true
}

variable "squad" {
  description = "squad label to apply to the service."
  type        = string
  default     = ""

  validation {
    condition     = !var.require_squad || var.squad != ""
    error_message = "squad needs to specified or disable check by setting require_squad = false"
  }
}

variable "zones" {
  default     = null
  description = "If specified, will spread nodes across these zones"
}

variable "subnetwork" {}

variable "master_ipv4_cidr_block" {
  description = "If specified, will use this CIDR block for the master's IP address"
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
