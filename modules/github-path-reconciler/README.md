# GitHub Path Reconciler Module

This module creates a GitHub path reconciliation system that monitors file paths in a GitHub repository and reconciles them when they change. It combines a regional-go-reconciler with both periodic (cron-based) and event-driven (push-based) reconciliation.

## Usage

```hcl
module "path-reconciler" {
  source = "chainguard-dev/terraform-infra-common//modules/github-path-reconciler"

  project_id     = var.project_id
  name           = "my-path-reconciler"
  primary-region = "us-central1"
  regions        = var.regions

  service_account = google_service_account.reconciler.email

  # Container configuration
  containers = {
    reconciler = {
      source = {
        working_dir = path.module
        importpath  = "./cmd/reconciler"
      }
      ports = [{
        container_port = 8080
      }]
      env = [{
        name  = "OCTO_IDENTITY"
        value = "my-reconciler"
      }]
    }
  }

  # Path patterns to match (with exactly one capture group each)
  path_patterns = [
    "^configs/(.+\\.yaml)$",  # Match YAML files in configs/
    "^deployments/(.+)$",      # Match everything in deployments/
  ]

  # Repository configuration
  github_owner      = "my-org"
  github_repo       = "my-repo"
  octo_sts_identity = "my-reconciler"

  # Event broker for push notifications
  broker = var.github_events_broker

  # Resync every 6 hours
  resync_period_hours = 6

  notification_channels = var.notification_channels
  squad                 = "platform"
  product               = "infrastructure"
}
```

## Features

- **Path Pattern Matching**: Define regex patterns to match specific file paths
- **Dual Reconciliation Modes**:
  - **Event-Driven**: Responds immediately to push events with high priority
  - **Periodic**: Full repository scan on a configurable schedule
- **Built-in Workqueue**: Integrated workqueue with priority support
- **Regional Deployment**: Deploy reconciler services across multiple regions
- **Pausable**: Single control to pause both cron and push listeners

## Architecture

The module creates:

1. **Reconciler Service** (via `regional-go-reconciler`):
   - Implements the workqueue service protocol
   - Processes path reconciliation requests
   - Deployed across all configured regions

2. **Cron Job** (periodic reconciliation):
   - Runs on a schedule (configurable in hours)
   - Fetches all files from the repository at HEAD
   - Matches files against path patterns
   - Enqueues matched paths with time-bucketed delays (priority 0)

3. **Push Listener** (event-driven reconciliation):
   - Subscribes to GitHub push events via CloudEvents
   - Compares commits to find changed files
   - Matches changed files against path patterns
   - Enqueues matched paths immediately (priority 100)

## Path Patterns

Path patterns are regular expressions with **exactly one capture group**. The captured portion becomes the path in the resource URL.

**Note:** Patterns are automatically anchored with `^` and `$`, ensuring full-path matching. Do not include these anchors in your patterns.

Examples:
```hcl
path_patterns = [
  # Match all files (entire path)
  "(.+)",

  # Match only YAML files (entire path)
  "(.+\\.yaml)",

  # Match files in a specific directory (entire path)
  "(infrastructure/.+)",
]
```

The module will create resource URLs in the format:
```
https://github.com/{owner}/{repo}/blob/{branch}/{captured_path}
```

## Reconciler Implementation

Your reconciler should implement the workqueue protocol. The key will be a GitHub URL to the file path:

```go
import (
    "github.com/chainguard-dev/terraform-infra-common/pkg/githubreconciler"
    "github.com/chainguard-dev/terraform-infra-common/pkg/workqueue"
)

func (r *Reconciler) Process(ctx context.Context, req *workqueue.ProcessRequest) (*workqueue.ProcessResponse, error) {
    log := clog.FromContext(ctx)

    // Parse the GitHub URL from the key
    res, err := githubreconciler.ParseResource(req.Key)
    if err != nil {
        return nil, err
    }

    log.Infof("Reconciling path: %s in %s/%s", res.Path, res.Owner, res.Repo)

    // Your reconciliation logic here
    // ...

    return &workqueue.ProcessResponse{}, nil
}
```

## Reconciliation Triggers

### Periodic (Cron)
- Runs every `resync_period_hours` (1-24 hours)
- Fetches complete repository tree at HEAD
- Uses time-bucketed delays to spread load across the period
- Priority: 0 (normal)

### Push Events
- Triggers on GitHub push events
- Uses `CompareCommits` API to get all changed files
- Handles all merge strategies (merge commits, squash, rebase)
- Priority: 100 (immediate)

## Safe Rollout Process

To safely deploy a new path reconciler, follow these steps:

1. **Initial Deployment** - Deploy with `paused = true` and `deletion_protection = false`:
   ```hcl
   module "my-reconciler" {
     # ... other configuration ...
     paused = true
     deletion_protection = false
   }
   ```

2. **Create Octo STS Identity** - After applying, use the service account's `unique_id` output to create the Octo STS identity in the GitHub organization. This grants the reconciler access to the GitHub API.

3. **Unpause** - Once the Octo STS identity is configured, set `paused = false` and apply:
   ```hcl
   paused = false
   ```

4. **Enable Protection** - After verifying the reconciler works correctly and you're confident you won't need to tear it down quickly, enable deletion protection:
   ```hcl
   deletion_protection = true
   ```

## Variables

See [variables.tf](./variables.tf) for all available configuration options.

Key variables:
- `path_patterns`: List of regex patterns (each with one capture group)
- `github_owner`, `github_repo`: Repository to monitor
- `octo_sts_identity`: Octo STS identity for GitHub authentication
- `broker`: Map of region to CloudEvents broker topic
- `resync_period_hours`: How often to run full reconciliation (1-24)
- `paused`: Pause both cron and push listeners
- `deletion_protection`: Prevent accidental deletion (disable during initial rollout)

<!-- BEGIN_TF_DOCS -->
## Requirements

No requirements.

## Providers

No providers.

## Modules

| Name | Source | Version |
|------|--------|---------|
| <a name="module_authorize-receiver-per-region"></a> [authorize-receiver-per-region](#module\_authorize-receiver-per-region) | ../authorize-private-service | n/a |
| <a name="module_cron"></a> [cron](#module\_cron) | ../cron | n/a |
| <a name="module_push-listener"></a> [push-listener](#module\_push-listener) | ../regional-go-service | n/a |
| <a name="module_push-subscription"></a> [push-subscription](#module\_push-subscription) | ../cloudevent-trigger | n/a |
| <a name="module_reconciler"></a> [reconciler](#module\_reconciler) | ../regional-go-reconciler | n/a |

## Resources

No resources.

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_broker"></a> [broker](#input\_broker) | A map from each of the input region names to the name of the Broker topic in that region. | `map(string)` | n/a | yes |
| <a name="input_concurrent-work"></a> [concurrent-work](#input\_concurrent-work) | The amount of concurrent work to dispatch at a given time. | `number` | `20` | no |
| <a name="input_containers"></a> [containers](#input\_containers) | The containers to run in the service.  Each container will be run in each region. | <pre>map(object({<br/>    source = object({<br/>      base_image  = optional(string, "cgr.dev/chainguard/static:latest-glibc@sha256:d4c20db9cb2dbf1ac9ec77f9dbc11080a78514a5f9b96096965550dbd1c73e09")<br/>      working_dir = string<br/>      importpath  = string<br/>      env         = optional(list(string), [])<br/>    })<br/>    args = optional(list(string), [])<br/>    ports = optional(list(object({<br/>      name           = optional(string, "h2c")<br/>      container_port = number<br/>    })), [])<br/>    resources = optional(<br/>      object(<br/>        {<br/>          limits = optional(object(<br/>            {<br/>              cpu    = string<br/>              memory = string<br/>            }<br/>          ), null)<br/>          cpu_idle          = optional(bool)<br/>          startup_cpu_boost = optional(bool, true)<br/>        }<br/>      ),<br/>      {}<br/>    )<br/>    env = optional(list(object({<br/>      name  = string<br/>      value = optional(string)<br/>      value_source = optional(object({<br/>        secret_key_ref = object({<br/>          secret  = string<br/>          version = string<br/>        })<br/>      }), null)<br/>    })), [])<br/>    regional-env = optional(list(object({<br/>      name  = string<br/>      value = map(string)<br/>    })), [])<br/>    regional-cpu-idle = optional(map(bool), {})<br/>    volume_mounts = optional(list(object({<br/>      name       = string<br/>      mount_path = string<br/>    })), [])<br/>    startup_probe = optional(object({<br/>      initial_delay_seconds = optional(number)<br/>      timeout_seconds       = optional(number, 240)<br/>      period_seconds        = optional(number, 240)<br/>      failure_threshold     = optional(number, 1)<br/>      tcp_socket = optional(object({<br/>        port = optional(number)<br/>      }), null)<br/>      grpc = optional(object({<br/>        port    = optional(number)<br/>        service = optional(string)<br/>      }), null)<br/>    }), null)<br/>    liveness_probe = optional(object({<br/>      initial_delay_seconds = optional(number)<br/>      timeout_seconds       = optional(number)<br/>      period_seconds        = optional(number)<br/>      failure_threshold     = optional(number)<br/>      http_get = optional(object({<br/>        path = optional(string)<br/>        http_headers = optional(list(object({<br/>          name  = string<br/>          value = string<br/>        })), [])<br/>      }), null)<br/>      grpc = optional(object({<br/>        port    = optional(number)<br/>        service = optional(string)<br/>      }), null)<br/>    }), null)<br/>  }))</pre> | `{}` | no |
| <a name="input_deletion_protection"></a> [deletion\_protection](#input\_deletion\_protection) | Whether to enable delete protection for the service. | `bool` | `true` | no |
| <a name="input_egress"></a> [egress](#input\_egress) | Which type of egress traffic to send through the VPC.<br/><br/>- ALL\_TRAFFIC sends all traffic through regional VPC network. This should be used if service is not expected to egress to the Internet.<br/>- PRIVATE\_RANGES\_ONLY sends only traffic to private IP addresses through regional VPC network | `string` | `"ALL_TRAFFIC"` | no |
| <a name="input_enable_profiler"></a> [enable\_profiler](#input\_enable\_profiler) | Enable continuous profiling for the service.  This has a small performance impact, which shouldn't matter for production services. | `bool` | `true` | no |
| <a name="input_execution_environment"></a> [execution\_environment](#input\_execution\_environment) | The execution environment for the service (options: EXECUTION\_ENVIRONMENT\_GEN1, EXECUTION\_ENVIRONMENT\_GEN2). | `string` | `"EXECUTION_ENVIRONMENT_GEN2"` | no |
| <a name="input_github_owner"></a> [github\_owner](#input\_github\_owner) | GitHub organization or user | `string` | n/a | yes |
| <a name="input_github_repo"></a> [github\_repo](#input\_github\_repo) | GitHub repository name | `string` | n/a | yes |
| <a name="input_labels"></a> [labels](#input\_labels) | Additional labels to add to all resources. | `map(string)` | `{}` | no |
| <a name="input_max-retry"></a> [max-retry](#input\_max-retry) | The maximum number of times a task will be retried before being moved to the dead-letter queue. Set to 0 for unlimited retries. | `number` | `100` | no |
| <a name="input_multi_regional_location"></a> [multi\_regional\_location](#input\_multi\_regional\_location) | The multi-regional location for the global workqueue bucket. Options: US, EU, ASIA. | `string` | `"US"` | no |
| <a name="input_name"></a> [name](#input\_name) | n/a | `string` | n/a | yes |
| <a name="input_notification_channels"></a> [notification\_channels](#input\_notification\_channels) | The channels to send notifications to. List of channel IDs | `list(string)` | `[]` | no |
| <a name="input_octo_sts_identity"></a> [octo\_sts\_identity](#input\_octo\_sts\_identity) | Octo STS identity for GitHub authentication | `string` | n/a | yes |
| <a name="input_otel_resources"></a> [otel\_resources](#input\_otel\_resources) | Resources to add to the OpenTelemetry resource. | `map(string)` | `{}` | no |
| <a name="input_path_patterns"></a> [path\_patterns](#input\_path\_patterns) | List of regex patterns with one capture group each for matching paths | `list(string)` | n/a | yes |
| <a name="input_paused"></a> [paused](#input\_paused) | Whether to pause both the cron service and push listener | `bool` | `false` | no |
| <a name="input_primary-region"></a> [primary-region](#input\_primary-region) | The primary region to run the cron job in | `string` | n/a | yes |
| <a name="input_product"></a> [product](#input\_product) | The product that this service belongs to. | `string` | `""` | no |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | n/a | `string` | n/a | yes |
| <a name="input_regional-volumes"></a> [regional-volumes](#input\_regional-volumes) | The volumes to make available to the containers in the service for mounting. | <pre>list(object({<br/>    name = string<br/>    gcs = optional(map(object({<br/>      bucket        = string<br/>      read_only     = optional(bool, true)<br/>      mount_options = optional(list(string), [])<br/>    })), {})<br/>    nfs = optional(map(object({<br/>      server    = string<br/>      path      = string<br/>      read_only = optional(bool, true)<br/>    })), {})<br/>  }))</pre> | `[]` | no |
| <a name="input_regions"></a> [regions](#input\_regions) | A map from region names to a network and subnetwork.  A service will be created in each region configured to egress the specified traffic via the specified subnetwork. | <pre>map(object({<br/>    network = string<br/>    subnet  = string<br/>  }))</pre> | n/a | yes |
| <a name="input_request_timeout_seconds"></a> [request\_timeout\_seconds](#input\_request\_timeout\_seconds) | The request timeout for the service in seconds. | `number` | `300` | no |
| <a name="input_resync_period_hours"></a> [resync\_period\_hours](#input\_resync\_period\_hours) | How often to resync all paths (in hours, must be between 1 and 24) | `number` | n/a | yes |
| <a name="input_scaling"></a> [scaling](#input\_scaling) | The scaling configuration for the service. | <pre>object({<br/>    min_instances                    = optional(number, 0)<br/>    max_instances                    = optional(number, 100)<br/>    max_instance_request_concurrency = optional(number, 1000)<br/>  })</pre> | `{}` | no |
| <a name="input_service_account"></a> [service\_account](#input\_service\_account) | The service account as which to run the reconciler service. | `string` | n/a | yes |
| <a name="input_slo"></a> [slo](#input\_slo) | Configuration for setting up SLO for the cloud run service | <pre>object({<br/>    enable          = optional(bool, false)<br/>    enable_alerting = optional(bool, false)<br/>    success = optional(object(<br/>      {<br/>        multi_region_goal = optional(number, 0.999)<br/>        per_region_goal   = optional(number, 0.999)<br/>      }<br/>    ), null)<br/>    monitor_gclb = optional(bool, false)<br/>  })</pre> | `{}` | no |
| <a name="input_team"></a> [team](#input\_team) | Team label to apply to resources (replaces deprecated 'squad'). | `string` | `""` | no |
| <a name="input_volumes"></a> [volumes](#input\_volumes) | The volumes to attach to the service. | <pre>list(object({<br/>    name = string<br/>    empty_dir = optional(object({<br/>      medium     = optional(string, "MEMORY")<br/>      size_limit = optional(string, "1Gi")<br/>    }), null)<br/>    csi = optional(object({<br/>      driver = string<br/>      volume_attributes = optional(object({<br/>        bucketName = string<br/>      }), null)<br/>    }), null)<br/>  }))</pre> | `[]` | no |
| <a name="input_workqueue_cpu_idle"></a> [workqueue\_cpu\_idle](#input\_workqueue\_cpu\_idle) | Set to false for a region in order to use instance-based billing for workqueue services (dispatcher and receiver). Defaults to true. To control reconciler cpu\_idle, use the 'regional-cpu-idle' field in the 'containers' variable. | `map(map(bool))` | <pre>{<br/>  "dispatcher": {},<br/>  "receiver": {}<br/>}</pre> | no |

## Outputs

No outputs.
<!-- END_TF_DOCS -->
