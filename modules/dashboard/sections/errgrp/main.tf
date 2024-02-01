variable "title" { type = string }
variable "collapsed" { default = false }
variable "project_id" { type = string }
variable "service_name" { type = string }

module "width" { source = "../width" }

module "errgrp" {
  source       = "../../widgets/errgrp"
  title        = var.title
  project_id   = var.project_id
  service_name = var.service_name
}

locals {
  tiles = [{
    yPos   = 0
    xPos   = 0,
    height = module.width.size,
    width  = module.width.size / 4,
    widget = module.errgrp.widget,
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
