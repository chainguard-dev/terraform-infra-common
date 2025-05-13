# Redis Module

Terraform module to create a GCP Redis instance within GCP.

## Features

- Creates a Redis instance with configurable tier and memory size
- Supports high availability configurations with replica instances
- Integrates with your existing VPC network for private connectivity
- Enables authentication and transit encryption options
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

TODO(mgreau): add inputs/outputs
