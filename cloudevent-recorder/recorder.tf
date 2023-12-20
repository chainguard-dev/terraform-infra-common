// Create an identity as which the recorder service will run.
resource "google_service_account" "recorder" {
  project = var.project_id

  account_id   = var.name
  display_name = "Cloudevents recorder"
  description  = "Dedicated service account for our recorder service."
}

// Grant the recorder service account permission to write to the regional GCS buckets.
resource "google_storage_bucket_iam_member" "recorder-writes-to-gcs-buckets" {
  for_each = var.regions

  bucket = google_storage_bucket.recorder[each.key].name
  role   = "roles/storage.admin"
  member = "serviceAccount:${google_service_account.recorder.email}"
}

resource "ko_build" "recorder-image" {
  base_image  = "cgr.dev/chainguard/static:latest-glibc"
  importpath  = "./cmd/recorder"
  working_dir = path.module
}

resource "cosign_sign" "recorder-image" {
  image = ko_build.recorder-image.image_ref
}

resource "ko_build" "logrotate-image" {
  base_image  = "cgr.dev/chainguard/static:latest-glibc"
  importpath  = "./cmd/logrotate"
  working_dir = path.module
}

resource "cosign_sign" "logrotate-image" {
  image = ko_build.logrotate-image.image_ref
}

resource "google_cloud_run_v2_service" "recorder-service" {
  for_each = var.regions

  provider = google-beta # For empty_dir
  project  = var.project_id
  name     = var.name
  location = each.key
  // This service should only be called by our Pub/Sub
  // subscription, so flag it as internal only.
  ingress = "INGRESS_TRAFFIC_INTERNAL_ONLY"

  launch_stage = "BETA"

  template {
    vpc_access {
      network_interfaces {
        network    = each.value.network
        subnetwork = each.value.subnet
      }
      egress = "ALL_TRAFFIC" // This should not egress
    }

    service_account = google_service_account.recorder.email
    containers {
      image = cosign_sign.recorder-image.signed_ref

      ports {
        container_port = 8080
      }

      env {
        name  = "LOG_PATH"
        value = "/logs"
      }
      volume_mounts {
        name       = "logs"
        mount_path = "/logs"
      }
    }
    containers {
      image = cosign_sign.logrotate-image.signed_ref

      env {
        name  = "BUCKET"
        value = google_storage_bucket.recorder[each.key].url
      }
      env {
        name  = "LOG_PATH"
        value = "/logs"
      }
      volume_mounts {
        name       = "logs"
        mount_path = "/logs"
      }
    }
    volumes {
      name = "logs"
      empty_dir {}
    }
  }
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

  depends_on = [google_cloud_run_v2_service.recorder-service]
  private-service = {
    region = each.value.region
    name   = google_cloud_run_v2_service.recorder-service[each.value.region].name
  }
}

module "recorder-dashboard" {
  source       = "../dashboard/cloudevent-receiver"
  service_name = var.name

  triggers = {
    for type, schema in var.types : "type: ${type}" => "${var.name}-${random_id.trigger-suffix[type].hex}"
  }
}
