# Valkey Module

Terraform module to create a Memorystore for Valkey instance within GCP.

This is the modern sibling of the [`redis`](../redis/) module. It is
deliberately opinionated where that module is configurable:

- **IAM auth only** (`IAM_AUTH`): clients connect as their workload identity
  under `roles/memorystore.dbConnectionUser`. There is no AUTH string, and so
  none of the Secret Manager plumbing the redis module carries. There is also
  no analog of the redis module's `authorized_client_editor_service_accounts`
  split: Valkey IAM has no data-plane read/write distinction —
  `dbConnectionUser` is the whole connect grant, and access control beyond it
  is in-band.
- **TLS always** (`SERVER_AUTHENTICATION`): clients verify the instance against
  the managed server CA, exposed as the `ca_pem` output.
- **Private Service Connect only**: the instance's PSC endpoints are
  auto-created in your VPC. There is no peering or reserved-range
  configuration.

Both instance shapes are first-class. `mode = "CLUSTER_DISABLED"` (the
default) is a standalone instance: one shard, a single primary endpoint
(`addr`) for standalone clients, and an optional reader endpoint
(`reader_addr`) when replicas are set. `mode = "CLUSTER"` shards across
`shard_count` nodes and `addr` becomes the discovery endpoint — clients must
speak the cluster protocol (e.g. `redis.ClusterClient`, not `redis.Client`).
Zone distribution defaults to `MULTI_ZONE`; pair standalone instances with
`SINGLE_ZONE` to co-locate with zonal clients.

## Prerequisites

PSC endpoint auto-creation requires a `gcp-memorystore` service connection
policy on the network. GCP allows exactly one per (network, region, service
class), so the policy is a network-level prerequisite the caller owns — once
per network and region, alongside the network itself rather than per instance:

```hcl
resource "google_project_service" "networkconnectivity" {
  project            = "my-project-id"
  service            = "networkconnectivity.googleapis.com"
  disable_on_destroy = false
}

resource "google_network_connectivity_service_connection_policy" "valkey" {
  # The network's project: the Shared VPC host project when the network is
  # shared, else the instance's own.
  project       = "my-project-id"
  name          = "valkey"
  location      = "us-central1"
  service_class = "gcp-memorystore"
  network       = module.networking.regional-networks["us-central1"].network
  psc_config {
    # Fully-qualified subnetwork id; PSC endpoints are created here.
    subnetworks = [module.networking.regional-networks["us-central1"].subnet-id]
  }

  depends_on = [google_project_service.networkconnectivity]
}
```

The policy lives in the network's project. On a Shared VPC that is the host
project — the instance stays in the service project (`project_id`), its
auto-created endpoints land there too, and `network` is the host project's
network id.

## Features

- Creates a Memorystore for Valkey instance in standalone (`CLUSTER_DISABLED`)
  or cluster mode, with configurable node type, shard count, replicas, and
  zone distribution
- Grants authorized client service accounts IAM connect access plus instance
  metadata read, so clients can resolve the endpoint and CA from the
  Memorystore API at boot instead of pinning them at apply time
- Exposes the single connect endpoint (`host`/`port`/`addr`) and the managed
  server CA bundle (`ca_pem`) for clients to pin
- Supports custom engine configs, maintenance windows, and RDB/AOF persistence
- Automatically enables the required Memorystore API
- Applies consistent squad/team labeling for resource organization and cost
  allocation

## Usage

```hcl
module "valkey" {
  source = "github.com/chainguard-dev/terraform-infra-common//modules/valkey"

  project_id = "my-project-id"
  name       = "my-valkey-instance"
  region     = "us-central1"
  team       = "platform-team"

  # The VPC the PSC endpoints land in (e.g. from the networking module); it
  # must already carry the gcp-memorystore connection policy (see above).
  network = module.networking.regional-networks["us-central1"].network

  node_type     = "STANDARD_SMALL"
  shard_count   = 1
  replica_count = 1

  engine_configs = {
    maxmemory-policy = "noeviction"
  }

  # Client workload identities granted the IAM connect role.
  authorized_client_service_accounts = [
    "my-service@my-project-id.iam.gserviceaccount.com",
  ]
}
```

Clients dial `module.valkey.addr` over TLS, verifying against
`module.valkey.ca_pem`, and authenticate with an IAM token for their service
account.

<!-- BEGIN_TF_DOCS -->
Copyright 2026 Chainguard, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

## Requirements

| Name | Version |
| ---- | ------- |
| <a name="requirement_google"></a> [google](#requirement\_google) | >= 7.34.0 |

## Providers

| Name | Version |
| ---- | ------- |
| <a name="provider_google"></a> [google](#provider\_google) | >= 7.34.0 |

## Modules

No modules.

## Resources

| Name | Type |
| ---- | ---- |
| [google_memorystore_instance.valkey](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/memorystore_instance) | resource |
| [google_project_iam_member.db_connection_user](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/project_iam_member) | resource |
| [google_project_iam_member.viewer](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/project_iam_member) | resource |
| [google_project_service.memorystore](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/project_service) | resource |

## Inputs

| Name | Description | Type | Default | Required |
| ---- | ----------- | ---- | ------- | :------: |
| <a name="input_authorized_client_service_accounts"></a> [authorized\_client\_service\_accounts](#input\_authorized\_client\_service\_accounts) | Service account emails granted roles/memorystore.dbConnectionUser (the IAM-auth connect grant) and roles/memorystore.viewer (instance metadata, for clients that resolve the endpoint and managed CA from the API at boot). Note these are project-level bindings: an authorized account can connect to any Memorystore instance in the project — so each SA must be listed by exactly one valkey module instance per project, or destroying one instance strips the grant the others rely on. Treat the list as append-only: entries are keyed by index, and removing or reordering a mid-list entry replaces the shifted grants, which can transiently (or, on an unlucky apply ordering, durably until re-apply) derole a surviving SA. | `list(string)` | `[]` | no |
| <a name="input_deletion_protection_enabled"></a> [deletion\_protection\_enabled](#input\_deletion\_protection\_enabled) | Whether the instance refuses deletion until this is unset. | `bool` | `true` | no |
| <a name="input_engine_configs"></a> [engine\_configs](#input\_engine\_configs) | Engine configuration parameters, e.g. { maxmemory-policy = "noeviction" }. | `map(string)` | `{}` | no |
| <a name="input_engine_version"></a> [engine\_version](#input\_engine\_version) | The version of Valkey software, e.g. VALKEY\_9\_0. | `string` | `"VALKEY_9_0"` | no |
| <a name="input_labels"></a> [labels](#input\_labels) | The resource labels to represent user-provided metadata. | `map(string)` | `{}` | no |
| <a name="input_maintenance_policy"></a> [maintenance\_policy](#input\_maintenance\_policy) | Maintenance policy for an instance. | <pre>object({<br/>    day = string<br/>    start_time = object({<br/>      hours   = optional(number, 0)<br/>      minutes = optional(number, 0)<br/>      seconds = optional(number, 0)<br/>      nanos   = optional(number, 0)<br/>    })<br/>  })</pre> | `null` | no |
| <a name="input_mode"></a> [mode](#input\_mode) | The instance mode. CLUSTER\_DISABLED serves standalone clients at a single primary endpoint; CLUSTER serves cluster-protocol clients at a discovery endpoint. | `string` | `"CLUSTER_DISABLED"` | no |
| <a name="input_name"></a> [name](#input\_name) | The instance ID, also used to name the service connection policy. | `string` | n/a | yes |
| <a name="input_network"></a> [network](#input\_network) | The VPC network (id or self link) the instance's PSC endpoints are created in. The network must already carry a gcp-memorystore service connection policy for the instance's region (see the README's prerequisites); GCP allows one per (network, region, service class), so the policy is caller-owned, not per-instance. | `string` | n/a | yes |
| <a name="input_node_type"></a> [node\_type](#input\_node\_type) | The machine type of each node. | `string` | `"STANDARD_SMALL"` | no |
| <a name="input_persistence_config"></a> [persistence\_config](#input\_persistence\_config) | Configuration of the persistence functionality. | <pre>object({<br/>    mode                = string<br/>    rdb_snapshot_period = optional(string)<br/>    aof_append_fsync    = optional(string)<br/>  })</pre> | <pre>{<br/>  "mode": "DISABLED"<br/>}</pre> | no |
| <a name="input_product"></a> [product](#input\_product) | Product label to apply to the service. | `string` | `"unknown"` | no |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | The ID of the project in which the resource belongs. | `string` | n/a | yes |
| <a name="input_region"></a> [region](#input\_region) | The GCP region to deploy resources to. | `string` | n/a | yes |
| <a name="input_replica_count"></a> [replica\_count](#input\_replica\_count) | The number of replica nodes per shard. | `number` | `1` | no |
| <a name="input_shard_count"></a> [shard\_count](#input\_shard\_count) | The number of shards. Must be 1 when mode is CLUSTER\_DISABLED. | `number` | `1` | no |
| <a name="input_team"></a> [team](#input\_team) | Team label to apply to resources (replaces deprecated 'squad'). | `string` | n/a | yes |
| <a name="input_zone_distribution"></a> [zone\_distribution](#input\_zone\_distribution) | Zone distribution of the instance's nodes. MULTI\_ZONE spreads nodes for availability; SINGLE\_ZONE places all nodes in the given zone, co-locating with zonal clients to cut cross-zone latency and egress. | <pre>object({<br/>    mode = string<br/>    zone = optional(string)<br/>  })</pre> | <pre>{<br/>  "mode": "MULTI_ZONE"<br/>}</pre> | no |

## Outputs

| Name | Description |
| ---- | ----------- |
| <a name="output_addr"></a> [addr](#output\_addr) | host:port of the instance's PSC connect endpoint. |
| <a name="output_ca_pem"></a> [ca\_pem](#output\_ca\_pem) | The managed server CA bundle (all currently-active CAs, concatenated PEM) clients pin to verify the instance's TLS connection. The CA is stable for years; only the leaf server cert rotates under it, so an apply-time capture stays valid across that rotation. The bundle is public certificate material. |
| <a name="output_host"></a> [host](#output\_host) | The IP address of the instance's PSC connect endpoint (the primary endpoint in CLUSTER\_DISABLED mode, the discovery endpoint in CLUSTER mode). |
| <a name="output_id"></a> [id](#output\_id) | Valkey instance ID. |
| <a name="output_port"></a> [port](#output\_port) | The port of the instance's PSC connect endpoint. |
| <a name="output_reader_addr"></a> [reader\_addr](#output\_reader\_addr) | host:port of the reader endpoint, for read-scaling standalone clients. Null unless mode is CLUSTER\_DISABLED with replicas. |
<!-- END_TF_DOCS -->
