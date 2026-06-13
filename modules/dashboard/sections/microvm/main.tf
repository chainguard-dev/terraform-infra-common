/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// The microvm section surfaces the two host-side observability loci of the
// microvm stack as two collapsible groups on a service/reconciler dashboard:
//
//   1. control plane — metrics the microvm.Manager records in this Cloud Run
//      service's process (filesystem ops on volumes it serves, credential
//      mints, endpoint requests, VM lifecycle, k8s bring-up phases). Scoped by
//      var.filter, the same service_name scoping the rest of the dashboard
//      uses.
//
//   2. agent pods — metrics the in-pod agent records on the GKE cluster
//      (egress decisions, the per-VM resource collector, qemu lifecycle).
//      Scoped to JUST var.namespace, the dedicated namespace this service's
//      agent pods run in, via the GMP prometheus_target resource label.
//
// The same metric name (e.g. microvm_fsop_total) is exported by both loci; the
// service_name (metric label) vs namespace (resource label) scoping keeps the
// two groups cleanly separated.

variable "filter" {
  description = "Monitoring filter scoping the control-plane metrics to this service (e.g. the service_name metric-label filter)."
  type        = list(string)
}

variable "namespace" {
  description = "The GKE namespace the service's microvm agent pods run in; the agent-pod group is scoped to it."
  type        = string
}

variable "collapsed" {
  description = "Whether the two microvm groups start collapsed."
  default     = true
}

module "width" { source = "../width" }

locals {
  // Agent pods are scraped by GMP PodMonitoring, so their namespace is a
  // prometheus_target resource label (not a metric label).
  pod_filter = ["resource.label.\"namespace\"=\"${var.namespace}\""]

  columns = 3
  unit    = module.width.size / local.columns
  col     = range(0, local.columns * local.unit, local.unit)
}

# ===== Control-plane group (this Cloud Run service's process) =====

module "cp_lifecycle" {
  source = "../../widgets/xy"
  title  = "VM lifecycle by state"
  filter = concat(var.filter, [
    "metric.type=\"prometheus.googleapis.com/microvm_lifecycle_state/gauge\"",
  ])
  group_by_fields = ["metric.label.\"state\""]
  primary_align   = "ALIGN_MEAN"
  primary_reduce  = "REDUCE_SUM"
  plot_type       = "STACKED_AREA"
}

module "cp_start_latency" {
  source = "../../widgets/xy"
  title  = "VM start latency by phase (P95)"
  filter = concat(var.filter, [
    "metric.type=\"prometheus.googleapis.com/microvm_start_phase_seconds/histogram\"",
    "metric.label.\"backend\"=\"k8s\"",
  ])
  group_by_fields = ["metric.label.\"phase\""]
  primary_align   = "ALIGN_DELTA"
  primary_reduce  = "REDUCE_PERCENTILE_95"
}

module "cp_token_mints" {
  source = "../../widgets/xy"
  title  = "Token mints by outcome"
  filter = concat(var.filter, [
    "metric.type=\"prometheus.googleapis.com/microvm_token_mint_total/counter\"",
  ])
  group_by_fields = ["metric.label.\"outcome\""]
  primary_align   = "ALIGN_RATE"
  primary_reduce  = "REDUCE_SUM"
}

module "cp_fsops" {
  source = "../../widgets/xy"
  title  = "Volume FS ops by op/result"
  filter = concat(var.filter, [
    "metric.type=\"prometheus.googleapis.com/microvm_fsop_total/counter\"",
  ])
  group_by_fields = ["metric.label.\"op\"", "metric.label.\"result\""]
  primary_align   = "ALIGN_RATE"
  primary_reduce  = "REDUCE_SUM"
  plot_type       = "STACKED_AREA"
}

module "cp_endpoint" {
  source = "../../widgets/xy"
  title  = "Endpoint requests by hostname/status"
  filter = concat(var.filter, [
    "metric.type=\"prometheus.googleapis.com/microvm_endpoint_requests_total/counter\"",
  ])
  group_by_fields = ["metric.label.\"hostname\"", "metric.label.\"code\""]
  primary_align   = "ALIGN_RATE"
  primary_reduce  = "REDUCE_SUM"
}

module "cp_credentials" {
  source = "../../widgets/xy"
  title  = "Credential reads by audience"
  filter = concat(var.filter, [
    "metric.type=\"prometheus.googleapis.com/microvm_credential_read_total/counter\"",
  ])
  group_by_fields = ["metric.label.\"audience\""]
  primary_align   = "ALIGN_RATE"
  primary_reduce  = "REDUCE_SUM"
}

# ===== Agent-pod group (scoped to var.namespace) =====

module "pod_egress" {
  source = "../../widgets/xy"
  title  = "Egress decisions by verdict/proto"
  filter = concat(local.pod_filter, [
    "metric.type=\"prometheus.googleapis.com/microvm_egress_total/counter\"",
  ])
  group_by_fields = ["metric.label.\"verdict\"", "metric.label.\"proto\""]
  primary_align   = "ALIGN_RATE"
  primary_reduce  = "REDUCE_SUM"
  plot_type       = "STACKED_AREA"
}

module "pod_blocked" {
  source = "../../widgets/xy"
  title  = "Netstack drops by reason"
  filter = concat(local.pod_filter, [
    "metric.type=\"prometheus.googleapis.com/microvm_netstack_blocked_total/counter\"",
  ])
  group_by_fields = ["metric.label.\"reason\""]
  primary_align   = "ALIGN_RATE"
  primary_reduce  = "REDUCE_SUM"
}

module "pod_cpu" {
  source = "../../widgets/xy"
  title  = "Guest CPU (cores)"
  filter = concat(local.pod_filter, [
    "metric.type=\"prometheus.googleapis.com/microvm_vm_cpu_seconds_total/counter\"",
  ])
  primary_align  = "ALIGN_RATE"
  primary_reduce = "REDUCE_SUM"
}

module "pod_memory" {
  source = "../../widgets/xy"
  title  = "Guest memory RSS (live)"
  filter = concat(local.pod_filter, [
    "metric.type=\"prometheus.googleapis.com/microvm_vm_memory_rss_bytes/gauge\"",
  ])
  primary_align  = "ALIGN_MEAN"
  primary_reduce = "REDUCE_SUM"
}

module "pod_scratch" {
  source = "../../widgets/xy"
  title  = "Guest scratch disk (live)"
  filter = concat(local.pod_filter, [
    "metric.type=\"prometheus.googleapis.com/microvm_vm_scratch_disk_bytes/gauge\"",
  ])
  primary_align  = "ALIGN_MEAN"
  primary_reduce = "REDUCE_SUM"
}

module "pod_exits" {
  source = "../../widgets/xy"
  title  = "VM exits by outcome"
  filter = concat(local.pod_filter, [
    "metric.type=\"prometheus.googleapis.com/microvm_vm_exit_total/counter\"",
  ])
  group_by_fields = ["metric.label.\"outcome\""]
  primary_align   = "ALIGN_RATE"
  primary_reduce  = "REDUCE_SUM"
}

locals {
  control_plane_tiles = [
    { yPos = local.unit, xPos = local.col[0], height = local.unit, width = local.unit, widget = module.cp_lifecycle.widget },
    { yPos = local.unit, xPos = local.col[1], height = local.unit, width = local.unit, widget = module.cp_start_latency.widget },
    { yPos = local.unit, xPos = local.col[2], height = local.unit, width = local.unit, widget = module.cp_token_mints.widget },
    { yPos = local.unit * 2, xPos = local.col[0], height = local.unit, width = local.unit, widget = module.cp_fsops.widget },
    { yPos = local.unit * 2, xPos = local.col[1], height = local.unit, width = local.unit, widget = module.cp_endpoint.widget },
    { yPos = local.unit * 2, xPos = local.col[2], height = local.unit, width = local.unit, widget = module.cp_credentials.widget },
  ]

  pod_tiles = [
    { yPos = local.unit, xPos = local.col[0], height = local.unit, width = local.unit, widget = module.pod_egress.widget },
    { yPos = local.unit, xPos = local.col[1], height = local.unit, width = local.unit, widget = module.pod_blocked.widget },
    { yPos = local.unit, xPos = local.col[2], height = local.unit, width = local.unit, widget = module.pod_exits.widget },
    { yPos = local.unit * 2, xPos = local.col[0], height = local.unit, width = local.unit, widget = module.pod_cpu.widget },
    { yPos = local.unit * 2, xPos = local.col[1], height = local.unit, width = local.unit, widget = module.pod_memory.widget },
    { yPos = local.unit * 2, xPos = local.col[2], height = local.unit, width = local.unit, widget = module.pod_scratch.widget },
  ]
}

module "control_plane" {
  source    = "../collapsible"
  title     = "microvm: control plane"
  tiles     = local.control_plane_tiles
  collapsed = var.collapsed
}

module "pods" {
  source    = "../collapsible"
  title     = "microvm: agent pods (${var.namespace})"
  tiles     = local.pod_tiles
  collapsed = var.collapsed
}

output "sections" {
  value = [module.control_plane.section, module.pods.section]
}
