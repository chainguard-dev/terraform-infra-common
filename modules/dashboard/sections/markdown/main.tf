variable "title" { type = string }
variable "content" { type = string }

module "width" { source = "../width" }

output "section" {
  value = [{
    yPos   = 0
    xPos   = 0
    height = 3
    width  = module.width.size,
    widget = {
      title = var.title
      text = {
        format  = "MARKDOWN"
        content = var.content
      }
    }
  }]
}
