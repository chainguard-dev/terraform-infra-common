# microvm Dashboard Section

Surfaces the [microvm](https://github.com/chainguard-dev/mono/tree/main/microvm)
stack's two host-side observability loci as two collapsible groups on a
service/reconciler dashboard. Enabled by setting `microvm = "<namespace>"` on
the `dashboard/service` module, where `<namespace>` is the dedicated GKE
namespace this service's microvm agent pods run in.

The two loci export some of the same metric names (e.g. `microvm_fsop_total`),
so the groups scope differently to keep them separate:

## microvm: control plane

Metrics the `microvm.Manager` records in **this Cloud Run service's process**,
scoped by `service_name` (the same scoping the rest of the dashboard uses).
This is the alert-grade locus — it runs outside any VM-escape blast radius.

- **VM lifecycle by state** — VMs reaching READY / TERMINATED / FAILED.
- **VM start latency by phase (P95)** — controller bring-up (`backend="k8s"`):
  pod create → ready → tunnel → channel → READY → ssh.
- **Token mints by outcome** — credential issuance decisions.
- **Volume FS ops by op/result** — every guest filesystem op dispatched on the
  controller, including `result="readonly_denied"` blocked writes.
- **Endpoint requests by status** — guest HTTP to host-provided endpoints.
- **Credential reads by audience** — guest reads of credfs-mounted tokens.

## microvm: agent pods (`<namespace>`)

Metrics the in-pod agent records on the GKE cluster, scoped to **just this
service's namespace** via the GMP `prometheus_target` resource label. This is
the agent-pod-trusted locus (inside the VM-escape blast radius) — fleet health
and evidence, not alert-grade.

- **Egress decisions by verdict/proto** — the userspace netstack's allow/deny.
- **Netstack drops by reason** — structural guards (sealed port, UDP drop, …).
- **VM exits by outcome** — clean vs abnormal QEMU exits.
- **Guest CPU (cores)** — from the on-demand `/proc` collector.
- **Guest memory RSS (live)** and **Guest scratch disk (live)** — current
  usage of the VMs live at scrape time.

## Trust

The control-plane group is alert-grade; the agent-pod group is evidentiary (an
escapee could forge or suppress it). See `microvm/ARCHITECTURE.md` for the full
trust taxonomy.

<!-- BEGIN_TF_DOCS -->
## Requirements

No requirements.

## Providers

No providers.

## Modules

| Name | Source | Version |
|------|--------|---------|
| <a name="module_control_plane"></a> [control\_plane](#module\_control\_plane) | ../collapsible | n/a |
| <a name="module_cp_credentials"></a> [cp\_credentials](#module\_cp\_credentials) | ../../widgets/xy | n/a |
| <a name="module_cp_endpoint"></a> [cp\_endpoint](#module\_cp\_endpoint) | ../../widgets/xy | n/a |
| <a name="module_cp_fsops"></a> [cp\_fsops](#module\_cp\_fsops) | ../../widgets/xy | n/a |
| <a name="module_cp_lifecycle"></a> [cp\_lifecycle](#module\_cp\_lifecycle) | ../../widgets/xy | n/a |
| <a name="module_cp_start_latency"></a> [cp\_start\_latency](#module\_cp\_start\_latency) | ../../widgets/xy | n/a |
| <a name="module_cp_token_mints"></a> [cp\_token\_mints](#module\_cp\_token\_mints) | ../../widgets/xy | n/a |
| <a name="module_pod_blocked"></a> [pod\_blocked](#module\_pod\_blocked) | ../../widgets/xy | n/a |
| <a name="module_pod_cpu"></a> [pod\_cpu](#module\_pod\_cpu) | ../../widgets/xy | n/a |
| <a name="module_pod_egress"></a> [pod\_egress](#module\_pod\_egress) | ../../widgets/xy | n/a |
| <a name="module_pod_exits"></a> [pod\_exits](#module\_pod\_exits) | ../../widgets/xy | n/a |
| <a name="module_pod_memory"></a> [pod\_memory](#module\_pod\_memory) | ../../widgets/xy | n/a |
| <a name="module_pod_scratch"></a> [pod\_scratch](#module\_pod\_scratch) | ../../widgets/xy | n/a |
| <a name="module_pods"></a> [pods](#module\_pods) | ../collapsible | n/a |
| <a name="module_width"></a> [width](#module\_width) | ../width | n/a |

## Resources

No resources.

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_collapsed"></a> [collapsed](#input\_collapsed) | Whether the two microvm groups start collapsed. | `bool` | `true` | no |
| <a name="input_filter"></a> [filter](#input\_filter) | Monitoring filter scoping the control-plane metrics to this service (e.g. the service\_name metric-label filter). | `list(string)` | n/a | yes |
| <a name="input_namespace"></a> [namespace](#input\_namespace) | The GKE namespace the service's microvm agent pods run in; the agent-pod group is scoped to it. | `string` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_sections"></a> [sections](#output\_sections) | n/a |
<!-- END_TF_DOCS -->
