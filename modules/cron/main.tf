terraform {
  required_providers {
    ko = {
      source = "ko-build/ko"
    }
    google = {
      source = "hashicorp/google"
    }
  }
}

resource "google_project_service" "cloud_run_api" {
  service = "run.googleapis.com"

  disable_on_destroy = false
}

resource "google_project_service" "cloudscheduler" {
  service = "cloudscheduler.googleapis.com"

  disable_on_destroy = false
}

locals {
  repo = var.repository != "" ? var.repository : "gcr.io/${var.project_id}/${var.name}"
}

resource "ko_build" "image" {
  importpath  = var.importpath
  working_dir = var.working_dir
  base_image  = var.base_image
  repo        = local.repo
}

resource "google_cloud_run_v2_job" "job" {
  name     = "${var.name}-cron"
  location = var.region

  # As Direct VPC is in BETA, we need to explicitly set the launch_stage to
  # BETA in order to use it.
  launch_stage = var.vpc_access != null ? "BETA" : null

  template {
    template {
      execution_environment = var.execution_environment
      service_account       = var.service_account
      max_retries           = var.max_retries
      timeout               = var.timeout
      containers {
        image = ko_build.image.image_ref

        dynamic "env" {
          for_each = var.env
          content {
            name  = env.key
            value = env.value
          }
        }

        dynamic "env" {
          for_each = var.secret_env
          content {
            name = env.key
            value_source {
              secret_key_ref {
                secret  = env.value
                version = "latest"
              }
            }
          }
        }
      }

      dynamic "vpc_access" {
        for_each = var.vpc_access[*]
        content {
          dynamic "network_interfaces" {
            for_each = vpc_access.value.network_interfaces[*]
            content {
              network    = network_interfaces.value.network
              subnetwork = network_interfaces.value.subnetwork
              tags       = network_interfaces.value.tags
            }
          }
          egress = vpc_access.value.egress
        }
      }
    }
  }
}

resource "google_service_account" "delivery" {
  project      = var.project_id
  account_id   = "${var.name}-dlv"
  display_name = "Dedicated service account for invoking ${google_cloud_run_v2_job.job.name}."
}

resource "google_cloud_run_v2_job_iam_binding" "authorize-calls" {
  project  = var.project_id
  location = google_cloud_run_v2_job.job.location
  name     = google_cloud_run_v2_job.job.name
  role     = "roles/run.invoker"
  members  = ["serviceAccount:${google_service_account.delivery.email}"]
}

resource "google_cloud_scheduler_job" "cron" {
  name     = "${var.name}-cron"
  schedule = var.schedule
  region   = var.region

  http_target {
    http_method = "POST"
    uri         = "https://${var.region}-run.googleapis.com/apis/run.googleapis.com/v1/namespaces/${var.project_id}/jobs/${google_cloud_run_v2_job.job.name}:run"

    oauth_token {
      service_account_email = google_service_account.delivery.email
    }
  }

}
