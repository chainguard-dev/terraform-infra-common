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

This module automatically manages Redis authentication credentials, when `auth_enabled = true`, the module will:
- Create a Redis instance with authentication enabled
- Retrieve the GCP-generated auth string
- Store the auth string in Secret Manager with Secret name automatically set to `{name}-auth` (e.g., "my-redis-instance-auth")
- The Secret Manager secret is created in the same project as the Redis instance
- Make the auth string available to authorized service accounts


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
|------|---------|
| <a name="requirement_google"></a> [google](#requirement\_google) | >= 4.79 |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_google"></a> [google](#provider\_google) | >= 4.79 |

## Modules

| Name | Source | Version |
|------|--------|---------|
| <a name="module_redis_auth_secret"></a> [redis\_auth\_secret](#module\_redis\_auth\_secret) | ../secret | n/a |

## Resources

| Name | Type |
|------|------|
| [google_project_iam_member.redis_client_sa](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/project_iam_member) | resource |
| [google_project_iam_member.redis_editor_sa](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/project_iam_member) | resource |
| [google_project_service.redis_api](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/project_service) | resource |
| [google_redis_instance.default](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/redis_instance) | resource |
| [google_secret_manager_secret_version.auth_string](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/secret_manager_secret_version) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_alternative_location_id"></a> [alternative\_location\_id](#input\_alternative\_location\_id) | The alternative zone where the instance will failover when zone is unavailable. | `string` | `""` | no |
| <a name="input_auth_enabled"></a> [auth\_enabled](#input\_auth\_enabled) | Indicates whether AUTH is enabled for the instance. | `bool` | `true` | no |
| <a name="input_authorized_client_editor_service_accounts"></a> [authorized\_client\_editor\_service\_accounts](#input\_authorized\_client\_editor\_service\_accounts) | List of service account emails that should be granted Redis editor (read-write) access | `list(string)` | `[]` | no |
| <a name="input_authorized_client_service_accounts"></a> [authorized\_client\_service\_accounts](#input\_authorized\_client\_service\_accounts) | List of service account emails that should be granted Redis viewer (read-only) access | `list(string)` | `[]` | no |
| <a name="input_authorized_network"></a> [authorized\_network](#input\_authorized\_network) | The full name of the Google Compute Engine network to which the instance is connected. Must be in the format: projects/{project\_id}/global/networks/{network\_name} | `string` | `""` | no |
| <a name="input_connect_mode"></a> [connect\_mode](#input\_connect\_mode) | The connection mode of the Redis instance. Valid values: DIRECT\_PEERING, PRIVATE\_SERVICE\_ACCESS. | `string` | `"PRIVATE_SERVICE_ACCESS"` | no |
| <a name="input_labels"></a> [labels](#input\_labels) | The resource labels to represent user-provided metadata. | `map(string)` | `{}` | no |
| <a name="input_maintenance_policy"></a> [maintenance\_policy](#input\_maintenance\_policy) | Maintenance policy for an instance. | <pre>object({<br/>    day = string<br/>    start_time = object({<br/>      hours   = number<br/>      minutes = number<br/>      seconds = number<br/>      nanos   = number<br/>    })<br/>  })</pre> | `null` | no |
| <a name="input_memory_size_gb"></a> [memory\_size\_gb](#input\_memory\_size\_gb) | Redis memory size in GiB. Minimum 1 GB, maximum 300 GB. | `number` | `1` | no |
| <a name="input_name"></a> [name](#input\_name) | The ID of the instance or a fully qualified identifier for the instance. | `string` | n/a | yes |
| <a name="input_notification_channels"></a> [notification\_channels](#input\_notification\_channels) | List of notification channels to alert. | `list(string)` | n/a | yes |
| <a name="input_persistence_config"></a> [persistence\_config](#input\_persistence\_config) | Configuration of the persistence functionality. | <pre>object({<br/>    persistence_mode    = string<br/>    rdb_snapshot_period = optional(string)<br/>  })</pre> | <pre>{<br/>  "persistence_mode": "DISABLED"<br/>}</pre> | no |
| <a name="input_product"></a> [product](#input\_product) | Product label to apply to the service. | `string` | `"unknown"` | no |
| <a name="input_product"></a> [product] | Product label to apply to the service. | `string` | `"unknown"` | no |
| <a name="input_project_id"></a> [project](#input\_project\_id) | The ID of the project in which the resource belongs. | `string` | n/a | yes |
| <a name="input_read_replicas_mode"></a> [read\_replicas\_mode](#input\_read\_replicas\_mode) | Read replicas mode. Can be: READ\_REPLICAS\_DISABLED or READ\_REPLICAS\_ENABLED. | `string` | `"READ_REPLICAS_DISABLED"` | no |
| <a name="input_redis_version"></a> [redis\_version](#input\_redis\_version) | The version of Redis software. | `string` | `"REDIS_7_2"` | no |
| <a name="input_region"></a> [region](#input\_region) | The GCP region to deploy resources to. | `string` | n/a | yes |
| <a name="input_replica_count"></a> [replica\_count](#input\_replica\_count) | The number of replica nodes. | `number` | `0` | no |
| <a name="input_reserved_ip_range"></a> [reserved\_ip\_range](#input\_reserved\_ip\_range) | The CIDR range of internal addresses that are reserved for this instance. | `string` | `null` | no |
| <a name="input_secret_accessor_sa_email"></a> [secret\_accessor\_sa\_email](#input\_secret\_accessor\_sa\_email) | The email of the service account that will access the secret. | `string` | n/a | yes |
| <a name="input_secret_version_adder"></a> [secret\_version\_adder](#input\_secret\_version\_adder) | The user allowed to populate new redis auth secret versions. | `string` | n/a | yes |
| <a name="input_squad"></a> [squad](#input\_squad) | Squad or team label applied to the instance (required). | `string` | `"unknown"` | no |
| <a name="input_tier"></a> [tier](#input\_tier) | The service tier of the instance. Valid values: BASIC, STANDARD\_HA. | `string` | `"STANDARD_HA"` | no |
| <a name="input_transit_encryption_mode"></a> [transit\_encryption\_mode](#input\_transit\_encryption\_mode) | The TLS mode of the Redis instance. Valid values: DISABLED, SERVER\_AUTHENTICATION. | `string` | `"SERVER_AUTHENTICATION"` | no |
| <a name="input_zone"></a> [zone](#input\_zone) | The zone where the instance will be deployed. | `string` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
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
| <a name="output_uri"></a> [uri](#output\_uri) | The connection URI to be used for accessing Redis. |
<!-- END_TF_DOCS -->
