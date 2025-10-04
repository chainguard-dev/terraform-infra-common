locals {
  # NB: substr returns the entire string if length is greater than the length of the string.
  prefix = substr(var.prefix, 0, (var.limit - length(var.suffix)))
}

output "result" {
  description = "The concatenation of prefix and suffix, with limit applied."
  value = "${local.prefix}${var.suffix}"
}
