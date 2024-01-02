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
