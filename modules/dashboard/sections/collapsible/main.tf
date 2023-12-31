variable "title" { type = string }
variable "tiles" {}
variable "collapsed" { default = false }

locals {
  start_row = length(var.tiles) == 0 ? 0 : min([for s in var.tiles : s.yPos]...)
}

module "width" { source = "../width" }

output "section" {
  value = concat([{
    yPos   = local.start_row
    xPos   = 0,
    height = length(var.tiles) == 0 ? 0 : max([for s in var.tiles : s.yPos + s.height - local.start_row]...),
    width  = module.width.size,
    widget = {
      title = var.title
      collapsibleGroup = {
        collapsed = var.collapsed
      }
    },
  }], var.tiles)
}
