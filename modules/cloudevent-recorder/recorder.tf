// Create an identity as which the recorder service will run.
resource "google_service_account" "recorder" {
  project = var.project_id

  account_id   = var.name
  display_name = "Cloudevents recorder"
  description  = "Dedicated service account for our recorder service."
}

// The recorder service account is the only identity that should be writing
// to the regional GCS buckets.
resource "google_storage_bucket_iam_binding" "recorder-writes-to-gcs-buckets" {
  for_each = var.regions

  bucket  = google_storage_bucket.recorder[each.key].name
  role    = "roles/storage.admin"
  members = ["serviceAccount:${google_service_account.recorder.email}"]
}

module "this" {
  source     = "../regional-go-service"
  project_id = var.project_id
  name       = var.name
  regions    = var.regions

  service_account = google_service_account.recorder.email
  containers = {
    "recorder" = {
      source = {
        working_dir = path.module
        importpath  = "./cmd/recorder"
      }
      ports = [{ container_port = 8080 }]
      env = [{
        name  = "LOG_PATH"
        value = "/logs"
      }]
      volume_mounts = [{
        name       = "logs"
        mount_path = "/logs"
      }]
    }
    "logrotate" = {
      source = {
        working_dir = path.module
        importpath  = "./cmd/logrotate"
      }
      env = [{
        name  = "LOG_PATH"
        value = "/logs"
      }]
      regional-env = [{
        name  = "BUCKET"
        value = { for k, v in google_storage_bucket.recorder : k => v.url }
      }]
      volume_mounts = [{
        name       = "logs"
        mount_path = "/logs"
      }]
    }
  }
  volumes = [{
    name      = "logs"
    empty_dir = {}
  }]
}

resource "random_id" "trigger-suffix" {
  for_each    = var.types
  byte_length = 2
}

// Create a trigger for each region x type that sends events to the recorder service.
module "triggers" {
  for_each = local.regional-types

  source = "../cloudevent-trigger"

  name       = "${var.name}-${random_id.trigger-suffix[each.value.type].hex}"
  project_id = var.project_id
  broker     = var.broker[each.value.region]
  filter     = { "type" : each.value.type }

  depends_on = [module.this]
  private-service = {
    region = each.value.region
    name   = var.name
  }
}

module "recorder-dashboard" {
  source       = "../dashboard/cloudevent-receiver"
  service_name = var.name

  labels = { for type, schema in var.types : replace(type, ".", "_") => "" }

  notification_channels = var.notification_channels

  triggers = {
    for type, schema in var.types : "type: ${type}" => {
      subscription_prefix   = "${var.name}-${random_id.trigger-suffix[type].hex}"
      alert_threshold       = schema.alert_threshold
      notification_channels = schema.notification_channels
    }
  }
}
