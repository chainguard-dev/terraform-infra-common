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

# ===== Repository-Level Token Metrics =====
# Note: These widgets use only bounded labels (repository, model, tool, turn, reconciler_type)
# to prevent cardinality explosion. Per-PR details are available via trace exemplars.
#
# Custom metrics (genai_token_prompt_total, genai_token_completion_total) are kept for
# backward compatibility. The gen_ai_client_token_usage histogram follows the
# OpenTelemetry GenAI semantic conventions (gen_ai.client.token.usage with gen_ai.token.type
# dimension) for compatibility with GenAI observability platforms.
# See: https://opentelemetry.io/docs/specs/semconv/gen-ai/gen-ai-metrics/

module "tokens_by_repo" {
  source = "../../widgets/xy"
  title  = "Token usage by repository (total)"
  filter = concat(var.filter, [
    "metric.type=\"prometheus.googleapis.com/genai_token_prompt_total/counter\"",
  ])

  group_by_fields = [
    "metric.label.\"repository\"",
  ]
  primary_align  = "ALIGN_DELTA"
  primary_reduce = "REDUCE_SUM"
  plot_type      = "STACKED_AREA"
}

module "tokens_by_model_repo" {
  source = "../../widgets/xy"
  title  = "Tokens by model per repository (total)"
  filter = concat(var.filter, [
    "metric.type=\"prometheus.googleapis.com/genai_token_prompt_total/counter\"",
  ])

  group_by_fields = [
    "metric.label.\"repository\"",
    "metric.label.\"model\"",
  ]
  primary_align  = "ALIGN_DELTA"
  primary_reduce = "REDUCE_SUM"
  plot_type      = "STACKED_AREA"
}

module "tool_calls_by_repo" {
  source = "../../widgets/xy"
  title  = "Tool calls per repository (total)"
  filter = concat(var.filter, [
    "metric.type=\"prometheus.googleapis.com/genai_tool_calls_total/counter\"",
  ])

  group_by_fields = [
    "metric.label.\"repository\"",
    "metric.label.\"tool\"",
  ]
  primary_align  = "ALIGN_DELTA"
  primary_reduce = "REDUCE_SUM"
  plot_type      = "STACKED_BAR"
}

module "tool_usage_breakdown" {
  source = "../../widgets/xy"
  title  = "Tool usage breakdown (total)"
  filter = concat(var.filter, [
    "metric.type=\"prometheus.googleapis.com/genai_tool_calls_total/counter\"",
  ])

  group_by_fields = [
    "metric.label.\"tool\"",
    "metric.label.\"model\"",
  ]
  primary_align  = "ALIGN_DELTA"
  primary_reduce = "REDUCE_SUM"
}

module "tokens_per_turn" {
  source = "../../widgets/xy"
  title  = "Tokens per agent turn (total)"
  filter = concat(var.filter, [
    "metric.type=\"prometheus.googleapis.com/genai_token_prompt_total/counter\"",
  ])

  group_by_fields = [
    "metric.label.\"turn\"",
    "metric.label.\"model\"",
  ]
  primary_align  = "ALIGN_DELTA"
  primary_reduce = "REDUCE_SUM"
  plot_type      = "LINE"
}

module "tokens_by_reconciler_type" {
  source = "../../widgets/xy"
  title  = "Tokens by reconciler type (total)"
  filter = concat(var.filter, [
    "metric.type=\"prometheus.googleapis.com/genai_token_prompt_total/counter\"",
  ])

  group_by_fields = [
    "metric.label.\"reconciler_type\"",
    "metric.label.\"model\"",
  ]
  primary_align  = "ALIGN_DELTA"
  primary_reduce = "REDUCE_SUM"
  plot_type      = "STACKED_AREA"
}

# ===== GenAI Semantic Convention Metrics (gen_ai.client.token.usage) =====
# These widgets use the OpenTelemetry GenAI semconv token usage histogram
# with gen_ai.token.type dimension (input/output).

module "semconv_tokens_by_repo" {
  source       = "../../widgets/xy-promql"
  title        = "GenAI semconv: token usage by repository"
  promql_query = "sum by (repository, gen_ai_token_type) (increase(gen_ai_client_token_usage_sum[10m]))"
  plot_type    = "STACKED_AREA"
}

module "semconv_tokens_by_model" {
  source       = "../../widgets/xy-promql"
  title        = "GenAI semconv: tokens by model"
  promql_query = "sum by (gen_ai_request_model, gen_ai_token_type) (increase(gen_ai_client_token_usage_sum[10m]))"
  plot_type    = "STACKED_AREA"
}

module "semconv_tokens_input_vs_output" {
  source       = "../../widgets/xy-promql"
  title        = "GenAI semconv: input vs output tokens"
  promql_query = "sum by (gen_ai_token_type) (increase(gen_ai_client_token_usage_sum[10m]))"
  plot_type    = "STACKED_BAR"
}

locals {
  columns = 3
  unit    = module.width.size / local.columns

  col = range(0, local.columns * local.unit, local.unit)

  tiles = [
    # Row 1: Original agent evaluation metrics
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

    # Row 2: Repository-level token metrics (bounded labels only)
    {
      yPos   = local.unit * 2,
      xPos   = local.col[0],
      height = local.unit,
      width  = local.unit,
      widget = module.tokens_by_repo.widget,
    },
    {
      yPos   = local.unit * 2,
      xPos   = local.col[1],
      height = local.unit,
      width  = local.unit,
      widget = module.tokens_by_model_repo.widget,
    },
    {
      yPos   = local.unit * 2,
      xPos   = local.col[2],
      height = local.unit,
      width  = local.unit,
      widget = module.tool_calls_by_repo.widget,
    },

    # Row 3: Tool usage, turn metrics, and reconciler type
    {
      yPos   = local.unit * 3,
      xPos   = local.col[0],
      height = local.unit,
      width  = local.unit,
      widget = module.tool_usage_breakdown.widget,
    },
    {
      yPos   = local.unit * 3,
      xPos   = local.col[1],
      height = local.unit,
      width  = local.unit,
      widget = module.tokens_per_turn.widget,
    },
    {
      yPos   = local.unit * 3,
      xPos   = local.col[2],
      height = local.unit,
      width  = local.unit,
      widget = module.tokens_by_reconciler_type.widget,
    },

    # Row 4: GenAI semantic convention metrics (gen_ai.client.token.usage)
    {
      yPos   = local.unit * 4,
      xPos   = local.col[0],
      height = local.unit,
      width  = local.unit,
      widget = module.semconv_tokens_by_repo.widget,
    },
    {
      yPos   = local.unit * 4,
      xPos   = local.col[1],
      height = local.unit,
      width  = local.unit,
      widget = module.semconv_tokens_by_model.widget,
    },
    {
      yPos   = local.unit * 4,
      xPos   = local.col[2],
      height = local.unit,
      width  = local.unit,
      widget = module.semconv_tokens_input_vs_output.widget,
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
