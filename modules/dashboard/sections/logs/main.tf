variable "title" { type = string }
variable "filter" { type = list(string) }
variable "collapsed" { default = true }
module "width" { source = "../width" }
variable "cloudrun_type" {
  type    = string
  default = "service"

  validation {
    condition     = contains(["service", "job"], var.cloudrun_type)
    error_message = "Allowed values for 'cloudrun_type' are 'service' or 'job'."
  }
}

locals {
  filter = concat(var.filter, var.cloudrun_type == "job" ? ["resource.type=\"cloud_run_job\""] : ["resource.type=\"cloud_run_revision\""])
}

module "logs" {
  source = "../../widgets/logs"
  title  = var.title
  filter = local.filter
}

locals {
  tiles = [{
    yPos   = 0
    xPos   = 0,
    height = module.width.size,
    width  = module.width.size,
    widget = module.logs.widget,
  }]
}

module "collapsible" {
  source = "../collapsible"

  title     = var.title
  tiles     = local.tiles
  collapsed = var.collapsed
}

output "section" {
  value = module.collapsible.section
}
