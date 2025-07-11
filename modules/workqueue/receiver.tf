// Compute a suffix that satisfies the regex:
// ^[a-z](?:[-a-z0-9]{4,28}[a-z0-9])$
resource "random_string" "receiver" {
  length  = 30 - length(local.sa_prefix)
  special = false
  upper   = false
}

// Create a dedicated GSA for the receiver service.
resource "google_service_account" "receiver" {
  project = var.project_id

  account_id   = "${local.sa_prefix}${random_string.receiver.result}"
  display_name = "Workqueue Receiver"
  description  = "The identity as which the workqueue receiver service runs for the ${var.name} workqueue."
}

// Stand up the receiver service in each of our regions.
module "receiver-service" {
  source        = "../regional-go-service"
  project_id    = var.project_id
  name          = "${var.name}-rcv"
  regions       = var.regions
  labels        = { "service" : "workqueue-receiver" }
  squad         = var.squad
  require_squad = var.require_squad

  deletion_protection = var.deletion_protection

  service_account = google_service_account.receiver.email
  containers = {
    "receiver" = {
      source = {
        working_dir = path.module
        importpath  = "github.com/chainguard-dev/terraform-infra-common/modules/workqueue/cmd/receiver"
      }
      resources = {
        limits = {
          memory = "4Gi"
          cpu    = "1000m"
        }
      }
      ports = [{ container_port = 8080 }]
      env = [
        {
          name  = "WORKQUEUE_MODE"
          value = "gcs"
        },
        {
          # The receiver doesn't use this, but the workqueue constructor wants it.
          name  = "WORKQUEUE_CONCURRENCY"
          value = "${var.concurrent-work}"
        },
      ]
      regional-env = [
        {
          name  = "WORKQUEUE_BUCKET"
          value = { for k, v in google_storage_bucket.workqueue : k => v.name }
        },
      ]
    }
  }

  notification_channels = var.notification_channels
}
