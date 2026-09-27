# Regional leasepool

Provisions one regional GCS bucket per region, containing any number of named
leases. Grants `roles/storage.objectUser` to exactly one supplied IAM identity via
additive IAM members. It creates no scheduler, service, or runtime lease objects.

```hcl
module "leases" {
  source     = "github.com/chainguard-dev/terraform-infra-common//modules/regional-leasepool"
  project_id = var.project_id
  name       = "maintenance-leases"
  regions    = local.regions
  identity   = google_service_account.worker.member
  team       = var.team
}
```

Pass `module.leases.buckets` as regional environment values to a service or cron;
use the corresponding bucket with `pkg/leasepool/gcs`. Distinct regions coordinate
independently. `identity` is a single `serviceAccount:`, `user:`, or `principal://`
IAM member, not a list or a public/group grant.

Buckets use uniform access, prohibit public access, and disable versioning and
soft delete. Frequent renewal must not accumulate historical lease records, and
restoring stale ownership is unsafe. There is no lifecycle rule deleting leases:
expiration belongs to the protocol. `force_destroy` defaults to false; stop all
participants before destroying or replacing a bucket.

Required inputs: `project_id`, `name`, `regions`, `identity`, `team`.
Optional inputs: `product`, `labels`, `force_destroy`.
Output: `buckets`, a map of region to bucket name ordered after IAM grants.

```sh
terraform init -backend=false
terraform test
```

Tests use mocked providers and create no cloud resources.
