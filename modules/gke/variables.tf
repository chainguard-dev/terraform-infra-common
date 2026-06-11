variable "name" {
  type = string
}

variable "service_account_suffix" {
  type        = string
  default     = null
  description = "Suffix appended to var.name to form the node service account ID. When null (the default), uses the region as the suffix (e.g. \"-us-central1\") so clusters sharing a project (e.g. the same name across regions) don't collide on this project-global resource. Set explicitly to pin a specific service account ID."
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



variable "team" {
  description = "Team label to apply to resources (replaces deprecated 'squad')."
  type        = string
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

variable "cluster_secondary_range_name" {
  description = "Name of an existing subnet secondary range to use for Pods. When null, GKE auto-allocates a pod range. Required on a Shared VPC, where secondary ranges are pre-created on the host subnet and referenced by name (GKE cannot create ranges on a subnet it does not own)."
  type        = string
  default     = null
}

variable "services_secondary_range_name" {
  description = "Name of an existing subnet secondary range to use for Services. When null, GKE auto-allocates a service range. Required on a Shared VPC, where secondary ranges are pre-created on the host subnet and referenced by name."
  type        = string
  default     = null
}


variable "pools" {
  type = map(object({
    min_node_count                    = optional(number, 1)
    max_node_count                    = optional(number, 1)
    machine_type                      = optional(string, "c3-standard-4")
    disk_type                         = optional(string, "pd-balanced")
    disk_size                         = optional(number, 100)
    ephemeral_storage_local_ssd_count = optional(number, 0)
    node_locations                    = optional(list(string), null)
    spot                              = optional(bool, false)
    gvisor                            = optional(bool, false)
    enable_nested_virtualization      = optional(bool, null)
    enable_secure_boot                = optional(bool, false)
    enable_integrity_monitoring       = optional(bool, true)
    labels                            = optional(map(string), {})
    tags                              = optional(list(string), [])
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
    guest_accelerator = optional(list(object({
      type  = string
      count = number
      gpu_driver_installation_config = optional(list(object({
        gpu_driver_version = string
      })), [])
    })), [])
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
  description = "Cluster autoscaling cpu limits"
}

variable "cluster_autoscaling_memory_limits" {
  type = object({
    resource_type = optional(string, "memory"),
    minimum       = optional(number, 8)
    maximum       = optional(number, 80)
  })
  default     = null
  description = "Cluster autoscaling memory limits"
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
  description = "Cluster autoscaling provisioning defaults"
}

variable "cluster_autoscaling_profile" {
  type        = string
  default     = null
  description = "Cluster autoscaling profile"
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
  default = {}
}

variable "advanced_datapath_observability_config" {
  description = "Config for Advanced Datapath Monitoring."
  type = object({
    enable         = optional(bool, false)
    enable_metrics = optional(bool, true)
    enable_relay   = optional(bool, true)
  })
  default = {}
}

variable "service_account_impersonation_email" {
  type        = string
  default     = null
  description = "Service account email impersonation for the service account created by this module."
}

variable "product" {
  description = "Product label to apply to the service."
  type        = string
  default     = "unknown"
}

variable "default_compute_class_enabled" {
  default     = false
  type        = bool
  description = "Specifies whether default compute class behavior is enabled. If enabled, cluster autoscaler will use Compute Class with name default for all the workloads, if not overridden."
}

variable "disable_horizontal_pod_autoscaling" {
  description = "Whether to disable the HPA addon on this cluster. When true, the metrics-server is not deployed and HPA resources will not function."
  type        = bool
  default     = null
}

variable "cluster_autoscaling_gpu_limits" {
  description = "GPU resource limits for cluster autoscaling NAP. Each entry specifies a GPU resource type and its min/max limits."
  type = list(object({
    resource_type = string
    minimum       = number
    maximum       = number
  }))
  default = []
}

variable "enable_fqdn_network_policy" {
  description = "Enable FQDN-based network policies on Dataplane V2 clusters. When true, the FQDNNetworkPolicy CRD (networking.gke.io/v1alpha1) becomes available for restricting pod egress to specific domains."
  type        = bool
  default     = false
}

variable "enable_dns_cache" {
  description = "Enable the NodeLocal DNSCache addon. Reduces external DNS lookup latency and load on kube-dns by caching responses on each node."
  type        = bool
  default     = false
}

variable "database_encryption_key" {
  description = "Cloud KMS key resource ID for application-layer Secrets encryption (e.g. projects/P/locations/L/keyRings/R/cryptoKeys/K). When null (default), Secrets use Google-managed etcd encryption only. The GKE service agent must hold roles/cloudkms.cryptoKeyEncrypterDecrypter on the key, and the key must outlive the cluster."
  type        = string
  default     = null
}
