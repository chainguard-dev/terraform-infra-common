variable "title" { type = string }
variable "collapsed" { default = false }
variable "alert" { type = string }

module "width" { source = "../width" }

module "alert" {
  source     = "../../widgets/alert"
  title      = "Alert"
  alert_name = var.alert
}

locals {
  tiles = [{
    yPos   = 0
    xPos   = 0
    height = 3
    width  = module.width.size
    widget = module.alert.widget
  }]
}

module "collapsible" {
  source = "../collapsible"

  // If no alert is defined, this is an empty collapsed section.
  title     = var.title
  tiles     = var.alert == "" ? [] : local.tiles
  collapsed = var.collapsed || var.alert == ""
}

output "section" {
  value = module.collapsible.section
}
