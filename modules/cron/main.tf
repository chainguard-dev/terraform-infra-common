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

module "audit-cronjob-serviceaccount" {
  source = "../audit-serviceaccount"

  project_id      = var.project_id
  service-account = var.service_account

  # The absence of authorized identities here means that
  # nothing is authorized to act as this service account.
  # Note: Cloud Run's usage doesn't show up in the
  # audit logs.

  notification_channels = var.notification_channels
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
      dynamic "volumes" {
        for_each = var.volumes
        content {
          name = volumes.value.name

          dynamic "secret" {
            for_each = volumes.value.secret != null ? { "" : volumes.value.secret } : {}
            content {
              secret = secret.value.secret
              dynamic "items" {
                for_each = secret.value.items
                content {
                  version = items.value.version
                  path    = items.value.path
                }
              }
            }
          }
        }
      }
      containers {
        image = ko_build.image.image_ref

        resources {
          limits = {
            cpu    = var.cpu
            memory = var.memory
          }
        }

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

        dynamic "volume_mounts" {
          for_each = var.volume_mounts
          content {
            name       = volume_mounts.value.name
            mount_path = volume_mounts.value.mount_path
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

module "audit-delivery-serviceaccount" {
  source = "../audit-serviceaccount"

  project_id      = var.project_id
  service-account = google_service_account.delivery.email

  # The absence of authorized identities here means that
  # nothing is authorized to act as this service account.
  # Note: Cloud Scheduler's usage doesn't show up in the
  # audit logs.

  notification_channels = var.notification_channels
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

// Get a project number for this project ID.
data "google_project" "project" { project_id = var.project_id }

// What identity is deploying this?
data "google_client_openid_userinfo" "me" {}

// Create an alert policy to notify if the job is accessed by an unauthorized entity.
resource "google_monitoring_alert_policy" "anomalous-job-access" {
  # In the absence of data, incident will auto-close after an hour
  alert_strategy {
    auto_close = "3600s"

    notification_rate_limit {
      period = "3600s" // re-alert hourly if condition still valid.
    }
  }

  display_name = "Abnormal CronJob Access: ${var.name}"
  combiner     = "OR"

  conditions {
    display_name = "Abnormal CronJob Access: ${var.name}"

    condition_matched_log {
      filter = <<EOT
      logName="projects/prod-enforce-fabc/logs/cloudaudit.googleapis.com%2Fdata_access"
      protoPayload.serviceName="run.googleapis.com"
      protoPayload.resourceName=("${join("\" OR \"", [
        "namespaces/${var.project_id}/jobs/${var.name}-cron",
        "projects/${var.project_id}/locations/${var.region}/jobs/${var.name}-cron",
      ])}")

      -- Allow CI to reconcile jobs and their IAM policies.
      -(
        protoPayload.authenticationInfo.principalEmail="${data.google_client_openid_userinfo.me.email}"
        protoPayload.methodName=("google.cloud.run.v2.Jobs.GetJob" OR "google.cloud.run.v2.Jobs.UpdateJob" OR "google.cloud.run.v2.Jobs.GetIamPolicy" OR "google.cloud.run.v2.Jobs.SetIamPolicy")
      )

      -- Allow the delivery service account to run the job.
      -(
        protoPayload.authenticationInfo.principalEmail="${google_service_account.delivery.email}"
        protoPayload.methodName="google.cloud.run.v1.Jobs.RunJob"
      )
      EOT
    }
  }

  # TODO(mattmoor): Enable notifications once this stabilizes.
  # notification_channels = var.notification_channels

  enabled = "true"
  project = var.project_id
}
