variable "text" { default = "Logs Explorer" }
variable "project" { type = string }
variable "params" { type = map(string) }

locals {
  params = urlencode(join("\n", [for key, value in var.params : "${key}=\"${value}\""]))
  link   = "https://console.cloud.google.com/logs/query;query=${local.params}?project=${var.project}"
}

output "markdown" {
  value = "[${var.text}](${local.link})"
}
