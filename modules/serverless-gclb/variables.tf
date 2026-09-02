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
A map from hostnames (managed by dns_zone), to the name of the regionalized cloud run service to which the hostname should be routed.  A managed SSL certificate will be created for each hostname (unless certificate_map is set), and a DNS record set will be created for each hostname pointing to the load balancer's global IP address.

external_managed_migration_state: The migration state for the load balancer, [PREPARE, TEST_BY_PERCENTAGE, and TEST_ALL_TRAFFIC].
external_managed_migration_testing_percentage: The percentage of traffic to route to new load balancer, [0, 100].
load_balancing_scheme: The default load balancing scheme to use.
EOF
  type = map(object({
    name                                          = string
    disabled                                      = optional(bool, false)
    external_managed_migration_state              = optional(string, null)
    external_managed_migration_testing_percentage = optional(number, null)
    load_balancing_scheme                         = optional(string, "EXTERNAL")
    connection_draining_timeout_sec               = optional(number, 300)
  }))
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

variable "certificate_map" {
  type        = string
  nullable    = false
  description = "Optional Certificate Manager certificate map id, formatted as \"//certificatemanager.googleapis.com/projects/.../certificateMaps/...\". When set, the HTTPS proxy serves TLS from this map and the module creates no per-hostname managed SSL certificates, escaping the 15-certificate-per-proxy limit (e.g. with a wildcard certificate). Create the map in an earlier apply than the one that sets this, so its id is known at plan time. When empty (the default), the module keeps its per-hostname managed-certificate behaviour. Migrating an existing proxy between the two modes is a two-apply operation, see retain_managed_certificates and the module README."
  default     = ""
}

variable "retain_managed_certificates" {
  type        = bool
  nullable    = false
  description = "Only meaningful when certificate_map is set. When true, the per-hostname managed SSL certificates are still created and stay attached to the HTTPS proxy alongside the certificate map, which is the legal intermediate state for migrating an existing proxy without a TLS gap. A target HTTPS proxy must always have >=1 SSL certificate or a certificate map, and the provider strips ssl_certificates before attaching the map, so flipping straight from certs to map in one apply is rejected (Error 412). Instead set certificate_map with this true in one apply (both attached), then set this back to false in a follow-up apply to drop the per-hostname certs. Roll back the same way in reverse, waiting for the recreated certs to be ACTIVE before removing the map. See the module README. Defaults to false, so greenfield proxies and existing callers are unaffected."
  default     = false
}
