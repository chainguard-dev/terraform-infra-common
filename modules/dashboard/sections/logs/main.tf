variable "title" { type = string }
variable "filter" { type = list(string) }
variable "collapsed" { default = true }

module "width" { source = "../width" }

module "logs" {
  source = "../../widgets/logs"
  title  = var.title
  filter = var.filter
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
