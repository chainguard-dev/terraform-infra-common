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

variable "labels" {
  description = "Labels to apply to the networking resources."
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

variable "hosted_zone_logging_enabled" {
  description = "Whether or not to enable DNS Hosted Zone Cloud Logging"
  type        = bool
  default     = true
}
