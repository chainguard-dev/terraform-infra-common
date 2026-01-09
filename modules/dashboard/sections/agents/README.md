# Agent Metrics Dashboard Section

Dashboard section for monitoring AI agent metrics including evaluation results and reconciler-level token/tool usage. Can be used with any agent service that emits these metrics.

## Features

### Original Agent Evaluation Metrics
- **Agent evaluation volume**: Rate of agent evaluations by tracer type and namespace
- **Agent evaluation failure rate**: Ratio of failed evaluations to total evaluations
- **Agent evaluation grade (P99)**: 99th percentile of evaluation grades

### Repository-Level Metrics
These metrics track agent behavior at the repository level, enabling cost tracking and performance analysis. Metrics use only bounded labels to prevent cardinality explosion. Per-PR details are available via trace exemplars in Cloud Trace.

- **Token usage by repository**: Total tokens consumed grouped by repository
- **Tokens by model per repository**: Token usage broken down by model for each repository
- **Tool calls per repository**: Number of tool invocations grouped by repository and tool type
- **Tool usage breakdown**: Distribution of tool calls across different tools and models
- **Tokens per agent turn**: Token consumption across multiple agent turns (useful for iterative agents)
- **Tokens by reconciler type**: Token usage comparing PR-based vs path-based reconcilers

## Metrics Used

### Agent Evaluations
- `prometheus.googleapis.com/agent_evaluations_total/counter`
- `prometheus.googleapis.com/agent_evaluation_failures_total/counter`
- `prometheus.googleapis.com/agent_evaluation_grade/gauge`

### Token & Tool Metrics
- `prometheus.googleapis.com/genai_token_prompt_total/counter` (with labels: `reconciler_type`, `repository`, `model`, `turn`)
- `prometheus.googleapis.com/genai_tool_calls_total/counter` (with labels: `reconciler_type`, `repository`, `tool`, `model`)

#### Label Definitions

**Bounded labels (used in metrics):**
- `repository`: Repository name extracted from reconciler_key (e.g., `chainguard-dev/enterprise-packages`)
- `reconciler_type`: Type of reconciler (`pr` or `path`)
- `model`: Model name (e.g., `claude-opus-4-1`, `gemini-3-pro-preview`)
- `tool`: Tool name (e.g., `git_clone`, `git_commit`)
- `turn`: Turn number, where 0 represents the first attempt (for multi-turn agents)

**Unbounded labels (available in traces only, not in metrics):**
- `reconciler_key`: Unique identifier for each reconciler instance (e.g., `pr:chainguard-dev/enterprise-packages/41025`)
  - Not included in metrics to prevent cardinality explosion
  - Available in traces for per-PR investigation
- `commit_sha`: Full 40-character git commit SHA
  - Not included in metrics to prevent cardinality explosion
  - Available in traces for per-commit investigation

**Accessing per-PR details:**
Dashboard widgets show aggregated metrics by repository. To investigate specific PRs, use Cloud Trace to view detailed execution traces with full `reconciler_key` and `commit_sha` information.

## Usage

Use this module with any agent service that emits `genai_token_*` and `genai_tool_calls_*` metrics. Customize the filter to target your specific agent service:

```hcl
module "agents" {
  source = "chainguard-dev/common/infra//modules/dashboard/sections/agents"

  title  = "Agent Metrics"
  filter = [
    "metric.label.\"service_name\"=\"your-agent-service-name\""
  ]
  collapsed = false
}
```

Replace `your-agent-service-name` with the actual service name of your agent that emits these metrics.

<!-- BEGIN_TF_DOCS -->
## Requirements

No requirements.

## Providers

No providers.

## Modules

| Name | Source | Version |
|------|--------|---------|
| <a name="module_collapsible"></a> [collapsible](#module\_collapsible) | ../collapsible | n/a |
| <a name="module_evaluation_failure_rate"></a> [evaluation\_failure\_rate](#module\_evaluation\_failure\_rate) | ../../widgets/xy-ratio | n/a |
| <a name="module_evaluation_grade_p99"></a> [evaluation\_grade\_p99](#module\_evaluation\_grade\_p99) | ../../widgets/xy | n/a |
| <a name="module_evaluation_volume"></a> [evaluation\_volume](#module\_evaluation\_volume) | ../../widgets/xy | n/a |
| <a name="module_tokens_by_model_repo"></a> [tokens\_by\_model\_repo](#module\_tokens\_by\_model\_repo) | ../../widgets/xy | n/a |
| <a name="module_tokens_by_reconciler_type"></a> [tokens\_by\_reconciler\_type](#module\_tokens\_by\_reconciler\_type) | ../../widgets/xy | n/a |
| <a name="module_tokens_by_repo"></a> [tokens\_by\_repo](#module\_tokens\_by\_repo) | ../../widgets/xy | n/a |
| <a name="module_tokens_per_turn"></a> [tokens\_per\_turn](#module\_tokens\_per\_turn) | ../../widgets/xy | n/a |
| <a name="module_tool_calls_by_repo"></a> [tool\_calls\_by\_repo](#module\_tool\_calls\_by\_repo) | ../../widgets/xy | n/a |
| <a name="module_tool_usage_breakdown"></a> [tool\_usage\_breakdown](#module\_tool\_usage\_breakdown) | ../../widgets/xy | n/a |
| <a name="module_width"></a> [width](#module\_width) | ../width | n/a |

## Resources

No resources.

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_collapsed"></a> [collapsed](#input\_collapsed) | n/a | `bool` | `false` | no |
| <a name="input_filter"></a> [filter](#input\_filter) | n/a | `list(string)` | n/a | yes |
| <a name="input_title"></a> [title](#input\_title) | n/a | `string` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_section"></a> [section](#output\_section) | n/a |
<!-- END_TF_DOCS -->
