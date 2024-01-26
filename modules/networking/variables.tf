variable "name" {
  type = string
}

variable "project_id" {
  type        = string
  description = "value of the project_id to use for the network"
}

variable "service_project_id" {
  type        = string
  description = "(optional) value of the project_id that hosts the Cloud Run services. Only needed if the service project is different from the project that hosts the network."
  default     = null
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

variable "network" {
  type        = string
  default     = null
  description = "(optional) The id of the network to create resources on. If omitted, a new network is created."
}
