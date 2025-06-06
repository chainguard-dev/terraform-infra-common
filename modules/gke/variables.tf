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
  type        = list(string)
}

variable "subnetwork" {
  description = "The subnetwork to deploy the cluster in."
  type        = string
}

variable "master_ipv4_cidr_block" {
  description = "If specified, will use this CIDR block for the master's IP address"
  type        = string
}

variable "pools" {
  type = map(object({
    min_node_count                    = optional(number, 1)
    max_node_count                    = optional(number, 1)
    machine_type                      = optional(string, "c3-standard-4")
    disk_type                         = optional(string, "pd-balanced")
    disk_size                         = optional(number, 100)
    ephemeral_storage_local_ssd_count = optional(number, 0)
    spot                              = optional(bool, false)
    gvisor                            = optional(bool, false)
    labels                            = optional(map(string), {})
    taints = optional(list(object({
      key    = string
      value  = string
      effect = string
    })), [])
    network_config = optional(object({
      enable_private_nodes = optional(bool, false)
      create_pod_range     = optional(bool, true)
      pod_ipv4_cidr_block  = optional(string, null)
    }), null)
  }))
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

variable "cluster_autoscaling" {
  type        = bool
  default     = false
  description = "Enabling of node auto-provisioning"
}

variable "cluster_autoscaling_cpu_limits" {
  type = object({
    resource_type = optional(string, "cpu")
    minimum       = optional(number, 4)
    maximum       = optional(number, 10)
  })
  default     = {}
  description = "cluster autoscaling cpu limits"
}

variable "cluster_autoscaling_memory_limits" {
  type = object({
    resource_type = optional(string, "memory"),
    minimum       = optional(number, 8)
    maximum       = optional(number, 80)
  })
  default     = null
  description = "cluster autoscaling memory limits"
}

variable "cluster_autoscaling_provisioning_defaults" {
  type = object({
    disk_size = optional(number, null)
    disk_type = optional(string, null)
    shielded_instance_config = optional(object({
      enable_secure_boot          = optional(bool, null)
      enable_integrity_monitoring = optional(bool, null)
    }), null)
    management = optional(object({
      auto_upgrade = optional(bool, null)
      auto_repair  = optional(bool, null)
    }), null)
  })
  default     = null
  description = "cluster autoscaling provisioning defaults"
}

variable "cluster_autoscaling_profile" {
  type        = string
  default     = null
  description = "cluster autoscaling profile"
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

variable "labels" {
  description = "Labels to apply to the gke resources."
  type        = map(string)
  default     = {}
}

variable "resource_usage_export_config" {
  description = "Config for exporting resource usage."
  type = object({
    bigquery_dataset_id                  = optional(string, "")
    enable_network_egress_metering       = optional(bool, false)
    enable_resource_consumption_metering = optional(bool, true)
  })
}
