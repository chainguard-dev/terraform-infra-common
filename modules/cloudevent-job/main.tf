terraform {
  required_providers {
    ko = { source = "ko-build/ko" }
  }
}

locals {
  job_regions = var.job.region != "" ? { var.job.region : var.regions[var.job.region] } : var.regions
}

resource "ko_build" "image" {
  importpath  = var.job.source.importpath
  working_dir = var.job.source.working_dir
  base_image  = var.job.source.base_image
}

resource "google_cloud_run_v2_job" "job" {
  provider = google-beta
  for_each = local.job_regions

  project  = var.project_id
  name     = var.name
  location = each.key

  deletion_protection = var.deletion_protection

  template {
    parallelism = var.parallelism
    task_count  = var.task_count
    labels      = merge(var.labels, { "squad" : var.squad })

    template {
      execution_environment = var.execution_environment
      service_account       = var.job.service_account
      max_retries           = var.max_retries
      timeout               = var.timeout
      containers {
        image = ko_build.image.image_ref

        resources {
          limits = {
            cpu    = var.cpu
            memory = var.memory
          }
        }
      }

      dynamic "containers" {
        for_each = var.enable_otel_sidecar ? [1] : []
        content {
          image = var.otel_collector_image
          // config via env is an option; https://pkg.go.dev/go.opentelemetry.io/collector/service#section-readme
          args = ["--config=env:OTEL_CONFIG"]
          env {
            name  = "OTEL_CONFIG"
            value = file("${path.module}/otel-config/config.yaml")
          }
        }
      }
    }
  }
}

resource "google_service_account" "invoker" {
  account_id   = "${var.name}-invoker"
  display_name = "${var.name} Job Invoker"
}

resource "google_cloud_run_v2_job_iam_binding" "invoker" {
  for_each = google_cloud_run_v2_job.job

  project  = var.project_id
  name     = each.value.name
  location = each.value.location

  role    = "roles/run.developer" // run.jobs.run and run.jobs.runWithOverrides
  members = [google_service_account.invoker.member]
}

module "invoker" {
  source = "../regional-go-service"

  project_id = var.project_id
  name       = "${var.job.name}-job-invoker"

  service_account = google_service_account.invoker.email

  regions              = local.job_regions
  squad                = var.squad
  otel_collector_image = var.otel_collector_image

  containers = {
    invoker = {
      source = {
        working_dir = path.module
        importpath  = "./cmd/invoker"
      }
      ports = [{ container_port = 8080 }]
      env = [{
        name = "JOB_NAME", value = var.job.name
      }]
      // If var.job.region is set, specify it for all regional services.
      // Otherwise, specify the region for each regional service.
      regional-env = [{
        name = "JOB_REGION"
        value = var.job.region == "" ? {
          for k, v in var.regions : k => k
          } : {
          for k, v in var.regions : k => var.job.region
        }
      }]
    }
  }

  notification_channels = var.notification_channels
}

module "trigger" {
  for_each   = var.regions
  depends_on = [module.invoker]
  source     = "../cloudevent-trigger"

  project_id = var.project_id
  name       = "${var.job.name}-job-trigger"

  broker = var.broker[each.key]

  private-service = {
    name   = "${var.job.name}-job-invoker"
    region = each.key
  }

  notification_channels = var.notification_channels
}
