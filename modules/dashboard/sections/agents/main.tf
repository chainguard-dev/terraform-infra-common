/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

variable "title" { type = string }
variable "filter" { type = list(string) }
variable "collapsed" { default = false }
module "width" { source = "../width" }

module "evaluation_volume" {
  source = "../../widgets/xy"
  title  = "Agent evaluation volume"
  filter = concat(var.filter, [
    "metric.type=\"prometheus.googleapis.com/agent_evaluations_total/counter\"",
  ])

  group_by_fields = [
    "metric.label.\"tracer_type\"",
    "metric.label.\"namespace\"",
  ]
  primary_align  = "ALIGN_RATE"
  primary_reduce = "REDUCE_SUM"
}

module "evaluation_failure_rate" {
  source = "../../widgets/xy-ratio"
  title  = "Agent evaluation failure rate"

  numerator_filter = concat(var.filter, [
    "metric.type=\"prometheus.googleapis.com/agent_evaluation_failures_total/counter\"",
  ])
  denominator_filter = concat(var.filter, [
    "metric.type=\"prometheus.googleapis.com/agent_evaluations_total/counter\"",
  ])

  numerator_group_by_fields = [
    "metric.label.\"tracer_type\"",
    "metric.label.\"namespace\"",
  ]
  denominator_group_by_fields = [
    "metric.label.\"tracer_type\"",
    "metric.label.\"namespace\"",
  ]

  numerator_align    = "ALIGN_RATE"
  numerator_reduce   = "REDUCE_SUM"
  denominator_align  = "ALIGN_RATE"
  denominator_reduce = "REDUCE_SUM"
}

module "evaluation_grade_p99" {
  source = "../../widgets/xy"
  title  = "Agent evaluation grade (P99)"
  filter = concat(var.filter, [
    "metric.type=\"prometheus.googleapis.com/agent_evaluation_grade/gauge\"",
  ])

  group_by_fields = [
    "metric.label.\"tracer_type\"",
    "metric.label.\"namespace\"",
  ]
  primary_align  = "ALIGN_MEAN"
  primary_reduce = "REDUCE_PERCENTILE_99"
}

locals {
  columns = 3
  unit    = module.width.size / local.columns

  col = range(0, local.columns * local.unit, local.unit)

  tiles = [
    {
      yPos   = local.unit,
      xPos   = local.col[0],
      height = local.unit,
      width  = local.unit,
      widget = module.evaluation_volume.widget,
    },
    {
      yPos   = local.unit,
      xPos   = local.col[1],
      height = local.unit,
      width  = local.unit,
      widget = module.evaluation_failure_rate.widget,
    },
    {
      yPos   = local.unit,
      xPos   = local.col[2],
      height = local.unit,
      width  = local.unit,
      widget = module.evaluation_grade_p99.widget,
    },
  ]
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
