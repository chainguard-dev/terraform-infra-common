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
