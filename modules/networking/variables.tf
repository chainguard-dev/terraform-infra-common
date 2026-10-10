variable "name" {
  type = string
}

variable "project_id" {
  type = string
}

variable "regions" {
  type        = list(string)
  description = "The list of regions in which to provision subnets suitable for use with Cloud Run direct VPC egress."
}

variable "cidr" {
  type    = string
  default = "10.0.0.0/8"
}

variable "netnum_offset" {
  type    = number
  default = 0
  validation {
    condition     = var.netnum_offset >= 0 && var.netnum_offset <= 255
    error_message = "value must be between 0 and 255"
  }
  description = "cidrsubnet netnum offset for the subnet. See https://developer.hashicorp.com/terraform/language/functions/cidrsubnet for more details"
}

variable "region_netnums" {
  type        = map(number)
  default     = {}
  description = "cidrsubnet netnums for specific regions' subnets, in place of netnum_offset plus the region's position in regions. A network adding a region names its netnum here to skip ranges it must not overlap, such as another network's it shares routes with, without renumbering the subnets it already has."

  validation {
    condition     = alltrue([for r in keys(var.region_netnums) : contains(var.regions, r)])
    error_message = "Every region in region_netnums must be in regions."
  }
  validation {
    condition     = alltrue([for n in values(var.region_netnums) : n >= 0 && n <= 255 && floor(n) == n])
    error_message = "Every region_netnums value must be an integer between 0 and 255."
  }
  validation {
    condition = length(distinct([
      for i, r in var.regions : lookup(var.region_netnums, r, var.netnum_offset + i)
    ])) == length(var.regions)
    error_message = "Two regions' subnets would share a netnum; give region_netnums values clear of netnum_offset plus each other region's position."
  }
}

variable "labels" {
  description = "Labels to apply to the networking resources."
  type        = map(string)
  default     = {}
}

variable "team" {
  description = "Team label to apply to resources (replaces deprecated 'squad')."
  type        = string
}

variable "product" {
  description = "Product label to apply to the service."
  type        = string
  default     = "unknown"
}

variable "hosted_zone_logging_enabled" {
  description = "Whether to enable Cloud DNS query logging on this network. Implemented via a DNS Server Policy attached to the VPC (the only mechanism that works for private managed zones)."
  type        = bool
  default     = false
}
