variable "name" {
  type = string
}

variable "project_id" {
  type = string
}

variable "regions" {
  description = "The set of regions containing backends for the load balancer (regions must be added here before they can be added as serving regions)."
  default     = ["us-central1"]
}

variable "serving_regions" {
  description = "The set of regions with backends suitable for serving traffic from the load balancer (regions must be removed from here before they can be removed from regions)."
  default     = ["us-central1"]
}

variable "dns_zone" {
  type        = string
  description = "The managed DNS zone in which to create record sets."
}

variable "public-services" {
  description = <<EOF
A map from hostnames (managed by dns_zone), to the name of the regionalized cloud run service to which the hostname should be routed.  A managed SSL certificate will be created for each hostname, and a DNS record set will be created for each hostname pointing to the load balancer's global IP address.

external_managed_migration_state: The migration state for the load balancer, [PREPARE, TEST_BY_PERCENTAGE, and TEST_ALL_TRAFFIC].
external_managed_migration_testing_percentage: The percentage of traffic to route to new load balancer, [0, 100].
load_balancing_scheme: The load balancing scheme to use (default EXTERNAL). Consumers already migrated to EXTERNAL_MANAGED must set this explicitly to preserve state.
EOF
  type = map(object({
    name                                          = string
    disabled                                      = optional(bool, false)
    external_managed_migration_state              = optional(string, null)
    external_managed_migration_testing_percentage = optional(number, null)
    load_balancing_scheme                         = optional(string, "EXTERNAL")
    connection_draining_timeout_sec               = optional(number, 300)
  }))

  validation {
    condition = alltrue([
      for v in values(var.public-services) : v.external_managed_migration_state == null || contains(["PREPARE", "TEST_BY_PERCENTAGE", "TEST_ALL_TRAFFIC"], v.external_managed_migration_state)
    ])
    error_message = "external_managed_migration_state must be one of PREPARE, TEST_BY_PERCENTAGE, TEST_ALL_TRAFFIC (or null)."
  }

  validation {
    condition = alltrue([
      for v in values(var.public-services) : v.external_managed_migration_testing_percentage == null || (v.external_managed_migration_testing_percentage >= 0 && v.external_managed_migration_testing_percentage <= 100)
    ])
    error_message = "external_managed_migration_testing_percentage must be between 0 and 100 (or null)."
  }

  validation {
    condition = alltrue([
      for v in values(var.public-services) : v.external_managed_migration_state != "TEST_BY_PERCENTAGE" || v.external_managed_migration_testing_percentage != null
    ])
    error_message = "external_managed_migration_testing_percentage is required when external_managed_migration_state is TEST_BY_PERCENTAGE."
  }
}

variable "notification_channels" {
  description = "The set of notification channels to which to send alerts."
  type        = list(string)
  default     = []
}

variable "iap" {
  description = "IAP configuration for the load balancer."
  type = object({
    oauth2_client_id     = optional(string, null)
    oauth2_client_secret = optional(string, null)
    enabled              = bool
  })
  default = null
}

variable "security-policy" {
  description = "The security policy associated with the backend service."
  type        = string
  default     = null
}



variable "team" {
  description = "team label to apply to the service."
  type        = string

}

variable "product" {
  description = "Product label to apply to the service."
  type        = string
  default     = "unknown"
}

variable "forwarding_rule_load_balancing" {
  type = object({
    external_managed_backend_bucket_migration_state              = optional(string, null)
    external_managed_backend_bucket_migration_testing_percentage = optional(number, null)
    load_balancing_scheme                                        = optional(string, "EXTERNAL")
  })
  default = {}
}

variable "enable_ipv6" {
  type        = bool
  description = "Enable dualstack ipv6+ipv4 support on the edge/public loadbalancer end point. When false (default), ipv4-only is deployed."
  default     = false
}
