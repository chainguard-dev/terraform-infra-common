// percent is an opinionated version of xy-ratio that assumes
// the numerator uses the same metric as the denominator
// with additional filtering.

variable "title" { type = string }
variable "legend" { default = "" }
variable "group_by_fields" { default = [] }
variable "numerator_additional_filter" { type = list(string) }
variable "common_filter" { type = list(string) }
variable "plot_type" { default = "LINE" }
variable "alignment_period" { default = "60s" }
variable "align" { default = "ALIGN_RATE" }
variable "reduce" { default = "REDUCE_SUM" }
variable "thresholds" { default = [] }

module "plot" {
  source = "../xy-ratio"

  title     = var.title
  legend    = var.legend
  plot_type = var.plot_type

  numerator_filter   = concat(var.common_filter, var.numerator_additional_filter)
  denominator_filter = var.common_filter

  alignment_period            = var.alignment_period
  thresholds                  = var.thresholds
  numerator_align             = var.align
  numerator_group_by_fields   = var.group_by_fields
  numerator_reduce            = var.reduce
  denominator_align           = var.align
  denominator_group_by_fields = var.group_by_fields
  denominator_reduce          = var.reduce
}

output "widget" {
  value = module.plot.widget
}
