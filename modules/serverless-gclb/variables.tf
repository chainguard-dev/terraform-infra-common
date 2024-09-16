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
  description = "A map from hostnames (managed by dns_zone), to the name of the regionalized cloud run service to which the hostname should be routed.  A managed SSL certificate will be created for each hostname, and a DNS record set will be created for each hostname pointing to the load balancer's global IP address."
  type = map(object({
    name     = string
    disabled = optional(bool, false)
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
    oauth2_client_id     = string
    oauth2_client_secret = string
    enabled              = optional(bool, true)
  })
  default = null
}
