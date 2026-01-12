# Dead-letter reenqueue Cloud Run Job
# This job allows manual reenqueuing of dead-lettered workqueue items.

locals {
  reenqueue_region = coalesce(var.primary-region, keys(var.regions)[0])
}

// Compute a suffix that satisfies the regex:
// ^[a-z](?:[-a-z0-9]{4,28}[a-z0-9])$
resource "random_string" "reenqueue" {
  length  = 30 - length(local.sa_prefix)
  special = false
  upper   = false
}

// Create a dedicated GSA for the reenqueue job.
resource "google_service_account" "reenqueue" {
  project = var.project_id

  account_id   = "${local.sa_prefix}${random_string.reenqueue.result}"
  display_name = "Workqueue Reenqueue Job"
  description  = "The identity as which the workqueue reenqueue job runs for the ${var.name} workqueue."
}

// Authorize the reenqueue service account to read/write the bucket (for Enumerate and Queue)
resource "google_storage_bucket_iam_member" "reenqueue-bucket-access" {
  bucket = google_storage_bucket.global-workqueue.name
  role   = "roles/storage.objectAdmin"
  member = "serviceAccount:${google_service_account.reenqueue.email}"
}

// The reenqueue cron job (paused by default, for manual invocation)
module "reenqueue" {
  source = "../cron"

  project_id      = var.project_id
  name            = "${var.name}-req"
  region          = local.reenqueue_region
  service_account = google_service_account.reenqueue.email

  importpath  = "github.com/chainguard-dev/terraform-infra-common/modules/workqueue/cmd/reenqueue"
  working_dir = path.module

  # Paused by default - this job is meant to be manually triggered
  paused   = true
  schedule = "0 0 * * *" # Placeholder, never runs when paused

  env = {
    "WORKQUEUE_MODE"        = "gcs"
    "WORKQUEUE_BUCKET"      = google_storage_bucket.global-workqueue.name
    "WORKQUEUE_CONCURRENCY" = var.concurrent-work
  }

  # VPC access using the reenqueue region's network configuration
  vpc_access = {
    network_interfaces = [{
      network    = var.regions[local.reenqueue_region].network
      subnetwork = var.regions[local.reenqueue_region].subnet
    }]
    egress = "ALL_TRAFFIC" // This should not egress
  }

  team                  = var.team
  product               = var.product
  notification_channels = var.notification_channels
  deletion_protection   = var.deletion_protection
}
