# Redis Module

Terraform module to create a Managed Redis instance within GCP.

## Features

- Creates a Redis instance with configurable tier and memory size
- Supports high availability configurations with replica instances
- Integrates with your existing VPC network for private connectivity
- Enables authentication and transit encryption options
- Automatically manages Redis AUTH credentials in Secret Manager
- Allows for custom maintenance windows
- Manages automatic backups with configurable snapshot periods
- Automatically enables the required Redis API
- Configures IAM permissions for authorized service accounts
- Applies consistent squad/team labeling for resource organization and cost allocation

## Usage

```hcl
module "redis" {
  source  = "github.com/chainguard-dev/terraform-infra-common//modules/redis"

  # Required parameters
  project_id      = "my-project-id"
  name            = "my-redis-instance"  # Required name for the instance
  region          = "us-central1"
  zone            = "us-central1-a"
  squad           = "platform-team"

  tier            = "STANDARD_HA"
  memory_size_gb  = 5

  alternative_location_id = "us-central1-c"

  # Network configuration - connect to existing VPC
  authorized_network = "projects/my-project-id/global/networks/my-vpc-network"

  # Auth configuration
  auth_enabled = true  # Auth credentials will automatically be stored in Secret Manager

  secret_accessor_sa_email = "my-service@my-project-id.iam.gserviceaccount.com"

  # Automated backups
  persistence_config = {
    persistence_mode    = "RDB"
    rdb_snapshot_period = "TWENTY_FOUR_HOURS"
  }

  # Configure a maintenance window
  # This schedules maintenance to occur on Tuesdays at 2:30 AM
  maintenance_policy = {
    day = "TUESDAY"
    start_time = {
      hours   = 2
      minutes = 30
      seconds = 0
      nanos   = 0
    }
  }

}
```

## Authentication Management

When `auth_enabled = true`, the module:
- Creates a Redis instance with authentication enabled
- Creates a Secret Manager secret named `{name}-auth` (e.g. "my-redis-instance-auth") in the instance's project
- Stores the GCP-generated AUTH string as a version of that secret, and stores a new one when the instance is replaced
- Grants `roles/secretmanager.secretAccessor` on the secret to `secret_accessor_sa_email` and the authorized client service accounts

Workloads read the AUTH string from Secret Manager at runtime: a Cloud Run
`secret_key_ref`, a GKE External Secret, or an application-side
`AccessSecretVersion` call. Never pass it through Terraform outputs or plain
environment variables.

### AUTH and plan identities

The secret version is written with `secret_data_wo`, a write-only argument:
the AUTH string never enters that resource's state, and refreshing the version
reads only its metadata. A plan of a root that uses this module therefore needs
no `secretmanager.versions.access` (`roles/secretmanager.secretAccessor`).

The instance itself is different. The Google provider calls
`redis.instances.getAuthString` on every read of an AUTH-enabled
`google_redis_instance`, resource or data source, and has no option to skip it;
a plan identity without the permission fails to refresh. The AUTH string is
therefore still in the state of the root that owns the instance (and of any
root with a `data "google_redis_instance"` lookup). Treat those roots as reading
a secret: their plan identity needs `redis.instances.getAuthString` and
`roles/redis.viewer` (no `roles/redis.admin`), and their state bucket is as
sensitive as the secret. Roots that only need the instance's host or port
should take them from somewhere other than a `google_redis_instance` lookup.

`ci_service_accounts` is unused and kept so existing callers still plan. CI
identities no longer get accessor on the secret, so a CI identity reading the
payload now trips the secret's anomaly alert, which is intended.

### Connecting without getAuthString

A root that only connects to an instance (it needs the address and CA, not the
password) should not look it up with `data "google_redis_instance"`, because
that read needs `getAuthString` and puts the AUTH string in the root's state.
When `auth_enabled = true`, the module publishes the connection details as
annotations on the `{name}-auth` secret:

| Annotation | Value |
| ---------- | ----- |
| `redis-host` | the instance's IP address |
| `redis-port` | its port |
| `redis-server-ca-pem` | the concatenated PEM of every current server CA; absent when transit encryption is off |

Read them with the secret data source, which fetches metadata only
(`secretmanager.secrets.get`, in `roles/secretmanager.viewer`), never the
payload:

```hcl
data "google_secret_manager_secret" "redis" {
  project   = "my-project-id"
  secret_id = "my-redis-instance-auth"
}

locals {
  redis_addr = "${data.google_secret_manager_secret.redis.annotations["redis-host"]}:${data.google_secret_manager_secret.redis.annotations["redis-port"]}"
  redis_ca   = lookup(data.google_secret_manager_secret.redis.annotations, "redis-server-ca-pem", "")
}
```

The annotations are updated in the same apply that changes the instance (a
replacement or a CA rotation); a consumer root picks the change up on its next
apply, as it would with the instance data source.

### Runbook: moving an existing instance onto the write-only version

Earlier versions of this module stored the AUTH string in
`google_secret_manager_secret_version.auth_string` with `secret_data`, so every
plan read the payload. The module now carries a
`removed { lifecycle { destroy = false } }` block for that resource. No state
surgery is needed: the first apply of a root after it picks up this module
version does the move.

1. **Review the plan.** For each instance (`<addr>` is the module's address in
   the root, for example `module.cache["us-central1"].module.redis`),
   the plan must show exactly:
   - `<addr>.google_secret_manager_secret_version.auth_string[0]` *will no
     longer be managed by Terraform* (forget, not destroy);
   - `<addr>.google_secret_manager_secret_version.auth[0]` *will be created*;
   - `<addr>.module.redis_auth_secret[0].google_secret_manager_secret_iam_binding.authorize-service-access`
     updated in place, dropping only the CI service accounts from `members`.

   Stop if the plan destroys any secret version or replaces the instance.
   This plan still refreshes the old version, so it reads the payload once,
   with the identity that already has access. Plans after the apply do not.
2. **Apply** through the normal provision workflow. Nothing to do by hand
   before or after; the old version stays enabled with the same value, so
   workloads keep connecting throughout.
3. **Verify the secret.** The newest version is enabled and the old one is
   still there:

   ```bash
   gcloud secrets versions list <name>-auth --project <project> --limit 3
   ```

   To confirm the stored value matches the instance without printing either
   (needs `getAuthString` and accessor, so run it as an operator, not a CI identity):

   ```bash
   diff \
     <(gcloud redis instances get-auth-string <name> --region <region> --project <project> --format='value(authString)' | tr -d '\n' | sha256sum) \
     <(gcloud secrets versions access latest --secret <name>-auth --project <project> | sha256sum) \
     && echo match
   ```

4. **Verify the workload.** Check the service's logs for Redis
   authentication errors (`NOAUTH`, `WRONGPASS`) after its next restart or
   rollout, and that its Redis-backed metrics (cache hit ratio, dashboards) are
   unchanged. Cloud Run and External Secrets resolve `latest` at start or
   refresh, so a value mismatch would show up then, not at apply.
5. **Confirm plans no longer read the payload.** Re-run the stage's plan: it
   should be a no-op, and Logs Explorer should show no `AccessSecretVersion`
   on `<name>-auth` from CI identities.
6. **Optional cleanup.** Once step 4 has held across a workload restart,
   disable the forgotten version (keep it for rollback; destroy it later):

   ```bash
   gcloud secrets versions disable <old-version> --secret <name>-auth --project <project>
   ```

If the plan shows the old version being *destroyed* rather than forgotten
(for example, a root pins this module at a copy without the `removed` block),
do not apply it. Forget the old resource from state first, then re-plan:

```bash
terraform state rm '<addr>.google_secret_manager_secret_version.auth_string[0]'
```

**Missing or wrong version.** If the secret has no enabled version, or step 3
reports a mismatch, add the current string out of band (as an
operator, since it needs `getAuthString` and version-adder):

```bash
gcloud redis instances get-auth-string <name> --region <region> --project <project> --format='value(authString)' \
  | tr -d '\n' \
  | gcloud secrets versions add <name>-auth --project <project> --data-file=-
```

A version added out of band is not tracked by Terraform and does not cause a
diff; workloads pick it up as `latest` at their next start or refresh. Never
paste the value into a terminal, ticket or chat.

**Rotation.** Memorystore issues a new AUTH string only when the instance is
replaced or AUTH is turned off and on. Replacement changes the instance's
`create_time`, which replaces `auth[0]` with the new string in the same apply.
Toggling AUTH in place does not: follow it with the out-of-band step above.

**Rollback.** Revert this module change. The revert must add
`removed { from = google_secret_manager_secret_version.auth  lifecycle { destroy = false } }`
to the module, so the apply forgets the write-only version instead of
destroying the version workloads are reading; the restored `auth_string`
resource then creates a new `secret_data` version with the same value. The
revert also restores CI accessor on the secret, which plans need again once
they read the payload.

<!-- BEGIN_TF_DOCS -->
Copyright 2025 Chainguard, Inc.

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
| <a name="requirement_terraform"></a> [terraform](#requirement\_terraform) | >= 1.11 |
| <a name="requirement_google"></a> [google](#requirement\_google) | >= 8.7.0 |

## Providers

| Name | Version |
| ---- | ------- |
| <a name="provider_google"></a> [google](#provider\_google) | >= 8.7.0 |

## Modules

| Name | Source | Version |
| ---- | ------ | ------- |
| <a name="module_redis_auth_secret"></a> [redis\_auth\_secret](#module\_redis\_auth\_secret) | ../secret | n/a |

## Resources

| Name | Type |
| ---- | ---- |
| [google_project_iam_member.redis_client_sa](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/project_iam_member) | resource |
| [google_project_iam_member.redis_editor_sa](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/project_iam_member) | resource |
| [google_project_service.redis_api](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/project_service) | resource |
| [google_redis_instance.default](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/redis_instance) | resource |
| [google_secret_manager_secret_version.auth](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/secret_manager_secret_version) | resource |
| [google_tags_location_tag_binding.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/tags_location_tag_binding) | resource |
| [google_project.resource_manager_tags](https://registry.terraform.io/providers/hashicorp/google/latest/docs/data-sources/project) | data source |

## Inputs

| Name | Description | Type | Default | Required |
| ---- | ----------- | ---- | ------- | :------: |
| <a name="input_alternative_location_id"></a> [alternative\_location\_id](#input\_alternative\_location\_id) | The alternative zone where the instance will failover when zone is unavailable. | `string` | `""` | no |
| <a name="input_auth_enabled"></a> [auth\_enabled](#input\_auth\_enabled) | Indicates whether AUTH is enabled for the instance. | `bool` | `true` | no |
| <a name="input_authorized_client_editor_service_accounts"></a> [authorized\_client\_editor\_service\_accounts](#input\_authorized\_client\_editor\_service\_accounts) | List of service account emails that should be granted Redis editor (read-write) access | `list(string)` | `[]` | no |
| <a name="input_authorized_client_service_accounts"></a> [authorized\_client\_service\_accounts](#input\_authorized\_client\_service\_accounts) | List of service account emails that should be granted Redis viewer (read-only) access | `list(string)` | `[]` | no |
| <a name="input_authorized_network"></a> [authorized\_network](#input\_authorized\_network) | The full name of the Google Compute Engine network to which the instance is connected. Must be in the format: projects/{project\_id}/global/networks/{network\_name} | `string` | `""` | no |
| <a name="input_ci_service_accounts"></a> [ci\_service\_accounts](#input\_ci\_service\_accounts) | Unused: the module writes the AUTH secret's version write-only, so refreshing it never reads the payload and CI identities need no access to it. Kept so existing callers still plan. | `list(string)` | `[]` | no |
| <a name="input_connect_mode"></a> [connect\_mode](#input\_connect\_mode) | The connection mode of the Redis instance. Valid values: DIRECT\_PEERING, PRIVATE\_SERVICE\_ACCESS. | `string` | `"PRIVATE_SERVICE_ACCESS"` | no |
| <a name="input_labels"></a> [labels](#input\_labels) | The resource labels to represent user-provided metadata. | `map(string)` | `{}` | no |
| <a name="input_maintenance_policy"></a> [maintenance\_policy](#input\_maintenance\_policy) | Maintenance policy for an instance. | <pre>object({<br/>    day = string<br/>    start_time = object({<br/>      hours   = number<br/>      minutes = number<br/>      seconds = number<br/>      nanos   = number<br/>    })<br/>  })</pre> | `null` | no |
| <a name="input_maxmemory_policy"></a> [maxmemory\_policy](#input\_maxmemory\_policy) | The maxmemory-policy Redis config: how the instance evicts keys under memory pressure, e.g. "noeviction". Null uses the Memorystore default. | `string` | `null` | no |
| <a name="input_memory_size_gb"></a> [memory\_size\_gb](#input\_memory\_size\_gb) | Redis memory size in GiB. Minimum 1 GB, maximum 300 GB. | `number` | `1` | no |
| <a name="input_name"></a> [name](#input\_name) | The ID of the instance or a fully qualified identifier for the instance. | `string` | n/a | yes |
| <a name="input_notification_channels"></a> [notification\_channels](#input\_notification\_channels) | List of notification channels to alert. | `list(string)` | n/a | yes |
| <a name="input_notify_keyspace_events"></a> [notify\_keyspace\_events](#input\_notify\_keyspace\_events) | The notify-keyspace-events Redis config: which keyspace notifications the instance emits, e.g. "Ex" for expired-key events. Null leaves notifications off. | `string` | `null` | no |
| <a name="input_persistence_config"></a> [persistence\_config](#input\_persistence\_config) | Configuration of the persistence functionality. | <pre>object({<br/>    persistence_mode    = string<br/>    rdb_snapshot_period = optional(string)<br/>  })</pre> | <pre>{<br/>  "persistence_mode": "DISABLED"<br/>}</pre> | no |
| <a name="input_product"></a> [product](#input\_product) | Unused: cost attribution comes from Resource Manager tags, not labels. Kept so existing callers still plan. | `string` | `null` | no |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | The ID of the project in which the resource belongs. | `string` | n/a | yes |
| <a name="input_read_replicas_mode"></a> [read\_replicas\_mode](#input\_read\_replicas\_mode) | Read replicas mode. Can be: READ\_REPLICAS\_DISABLED or READ\_REPLICAS\_ENABLED. | `string` | `"READ_REPLICAS_DISABLED"` | no |
| <a name="input_redis_version"></a> [redis\_version](#input\_redis\_version) | The version of Redis software. | `string` | `"REDIS_7_2"` | no |
| <a name="input_region"></a> [region](#input\_region) | The GCP region to deploy resources to. | `string` | n/a | yes |
| <a name="input_replica_count"></a> [replica\_count](#input\_replica\_count) | The number of replica nodes. | `number` | `0` | no |
| <a name="input_reserved_ip_range"></a> [reserved\_ip\_range](#input\_reserved\_ip\_range) | The CIDR range of internal addresses that are reserved for this instance. | `string` | `null` | no |
| <a name="input_resource_manager_tags"></a> [resource\_manager\_tags](#input\_resource\_manager\_tags) | Resource Manager tags to bind to the Redis instance, as tagKeys/<id> => tagValues/<id>. | `map(string)` | `{}` | no |
| <a name="input_secret_accessor_sa_email"></a> [secret\_accessor\_sa\_email](#input\_secret\_accessor\_sa\_email) | The email of the service account that will access the secret. | `string` | n/a | yes |
| <a name="input_secret_version_adder"></a> [secret\_version\_adder](#input\_secret\_version\_adder) | The user allowed to populate new redis auth secret versions. | `string` | n/a | yes |
| <a name="input_secret_version_adders"></a> [secret\_version\_adders](#input\_secret\_version\_adders) | Additional users allowed to populate new redis auth secret versions (e.g., Terraform service accounts). | `list(string)` | `[]` | no |
| <a name="input_team"></a> [team](#input\_team) | Team label to apply to resources (replaces deprecated 'squad'). | `string` | n/a | yes |
| <a name="input_tier"></a> [tier](#input\_tier) | The service tier of the instance. Valid values: BASIC, STANDARD\_HA. | `string` | `"STANDARD_HA"` | no |
| <a name="input_transit_encryption_mode"></a> [transit\_encryption\_mode](#input\_transit\_encryption\_mode) | The TLS mode of the Redis instance. Valid values: DISABLED, SERVER\_AUTHENTICATION. | `string` | `"SERVER_AUTHENTICATION"` | no |
| <a name="input_zone"></a> [zone](#input\_zone) | The zone for the instance's primary node. Leave unset to let Memorystore choose a zone in the region. | `string` | `null` | no |

## Outputs

| Name | Description |
| ---- | ----------- |
| <a name="output_auth_enabled"></a> [auth\_enabled](#output\_auth\_enabled) | Whether AUTH is enabled for the Redis instance |
| <a name="output_auth_secret_id"></a> [auth\_secret\_id](#output\_auth\_secret\_id) | The ID of the Secret Manager secret containing the Redis AUTH string |
| <a name="output_connection_name"></a> [connection\_name](#output\_connection\_name) | The connection name of the instance to be used in connection strings. |
| <a name="output_current_location_id"></a> [current\_location\_id](#output\_current\_location\_id) | The zone where the instance is currently located. |
| <a name="output_host"></a> [host](#output\_host) | The IP address of the instance. |
| <a name="output_id"></a> [id](#output\_id) | Redis instance ID. |
| <a name="output_memory_size_gb"></a> [memory\_size\_gb](#output\_memory\_size\_gb) | Redis memory size in GiB. |
| <a name="output_persistence_mode"></a> [persistence\_mode](#output\_persistence\_mode) | The persistence mode of the Redis instance. |
| <a name="output_port"></a> [port](#output\_port) | The port number of the instance. |
| <a name="output_rdb_snapshot_period"></a> [rdb\_snapshot\_period](#output\_rdb\_snapshot\_period) | The snapshot period for RDB persistence. |
| <a name="output_redis_version"></a> [redis\_version](#output\_redis\_version) | The version of Redis software. |
| <a name="output_region"></a> [region](#output\_region) | The region the instance lives in. |
| <a name="output_server_ca_cert"></a> [server\_ca\_cert](#output\_server\_ca\_cert) | The server CA certificate for the Redis instance |
| <a name="output_tag_binding_count"></a> [tag\_binding\_count](#output\_tag\_binding\_count) | Number of Resource Manager tag bindings on the Redis instance |
| <a name="output_uri"></a> [uri](#output\_uri) | The connection URI to be used for accessing Redis. |
<!-- END_TF_DOCS -->
