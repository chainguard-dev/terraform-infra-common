// Copyright 2026 Chainguard, Inc.
// SPDX-License-Identifier: Apache-2.0

terraform {
  required_providers {
    ko     = { source = "ko-build/ko" }
    cosign = { source = "chainguard-dev/cosign" }
  }
}

locals {
  default_labels = {
    basename(abspath(path.module)) = var.name
    terraform-module               = basename(abspath(path.module))
  }
  squad_label   = { squad = var.team, team = var.team }
  product_label = var.product != "unknown" ? { product = var.product } : {}
  merged_labels = merge(local.default_labels, local.squad_label, local.product_label, var.labels)

  // Collect any METRICS_PORT overrides declared in container env vars,
  // mirroring the same logic used in regional-service.
  extra_metrics_ports = toset(flatten([
    for c in values(var.containers) : [
      for e in c.env : e.value
      if e.name == "METRICS_PORT"
    ]
  ]))
  metrics_targets = join(", ", [
    for t in distinct(concat(["localhost:2112"], [for p in local.extra_metrics_ports : "localhost:${p}"])) : "\"${t}\""
  ])
}

// Build each application container image from source, mirroring regional-go-service.
resource "ko_build" "this" {
  for_each    = var.containers
  base_image  = each.value.source.base_image
  working_dir = each.value.source.working_dir
  importpath  = each.value.source.importpath
  env         = each.value.source.env
}

resource "cosign_sign" "this" {
  for_each = var.containers
  image    = ko_build.this[each.key].image_ref
  conflict = "REPLACE"
}

// Dedicated service account for Cloud Scheduler to invoke the jobs.
module "invoker_name" {
  source = "../limited-concat"
  prefix = var.name
  suffix = "-inv"
  limit  = 30
}

resource "google_service_account" "invoker" {
  project      = var.project_id
  account_id   = module.invoker_name.result
  display_name = "Cron invoker for ${var.name}"
}

// One Cloud Run Job per region.
resource "google_cloud_run_v2_job" "this" {
  for_each = var.regions
  provider = google-beta
  project  = var.project_id

  name         = var.name
  location     = each.key
  labels       = local.merged_labels
  launch_stage = var.launch_stage

  deletion_protection = var.deletion_protection

  template {
    task_count  = var.task_count
    parallelism = var.parallelism
    labels      = local.merged_labels

    template {
      execution_environment = var.execution_environment
      service_account       = var.service_account
      max_retries           = var.max_retries
      timeout               = var.timeout

      dynamic "volumes" {
        for_each = var.volumes
        content {
          name = volumes.value.name
          dynamic "empty_dir" {
            for_each = volumes.value.empty_dir != null ? { "" : volumes.value.empty_dir } : {}
            content {
              medium     = empty_dir.value.medium
              size_limit = empty_dir.value.size_limit
            }
          }
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
          dynamic "nfs" {
            for_each = volumes.value.nfs != null ? { "" : volumes.value.nfs } : {}
            content {
              server    = nfs.value.server
              path      = nfs.value.path
              read_only = nfs.value.read_only
            }
          }
          dynamic "gcs" {
            for_each = volumes.value.gcs != null ? { "" : volumes.value.gcs } : {}
            content {
              bucket        = gcs.value.bucket
              read_only     = gcs.value.read_only
              mount_options = gcs.value.mount_options
            }
          }
        }
      }

      // User-provided containers. Ports, probes, and cpu_idle are present
      // in the type for regional-go-service compatibility but are not used here.
      dynamic "containers" {
        for_each = var.containers
        content {
          image   = cosign_sign.this[containers.key].signed_ref
          command = length(containers.value.command) > 0 ? containers.value.command : null
          args    = length(containers.value.args) > 0 ? containers.value.args : null

          dynamic "resources" {
            for_each = containers.value.resources.limits != null ? [containers.value.resources] : []
            content {
              limits = resources.value.limits
            }
          }

          dynamic "env" {
            for_each = containers.value.env
            content {
              name  = env.value.name
              value = env.value.value
              dynamic "value_source" {
                for_each = env.value.value_source != null ? { "" : env.value.value_source } : {}
                content {
                  secret_key_ref {
                    secret  = value_source.value.secret_key_ref.secret
                    version = value_source.value.secret_key_ref.version
                  }
                }
              }
            }
          }

          // Regional env: select the value for this job's region.
          dynamic "env" {
            for_each = { for re in containers.value.regional-env : re.name => re.value[each.key] if contains(keys(re.value), each.key) }
            content {
              name  = env.key
              value = env.value
            }
          }

          dynamic "volume_mounts" {
            for_each = containers.value.volume_mounts
            content {
              name       = volume_mounts.value.name
              mount_path = volume_mounts.value.mount_path
            }
          }
        }
      }

      // OTel sidecar for metrics collection.
      dynamic "containers" {
        for_each = var.enable_otel_sidecar ? [1] : []
        content {
          image = var.otel_collector_image
          args  = ["--config=env:OTEL_CONFIG"]
          env {
            name = "OTEL_CONFIG"
            value = replace(replace(replace(replace(file("${path.module}/otel-config/config.yaml"),
              "REPLACE_ME_TEAM", var.team),
              "REPLACE_ME_PROJECT_ID", var.project_id),
              "REPLACE_ME_NAME", var.name),
            "REPLACE_ME_TARGETS", local.metrics_targets)
          }
        }
      }

      vpc_access {
        network_interfaces {
          network    = each.value.network
          subnetwork = each.value.subnet
        }
        egress = var.egress
      }
    }
  }

  lifecycle {
    ignore_changes = [
      template[0].template[0].containers[0].name,
      template[0].template[0].containers[1].name,
      template[0].template[0].containers[2].name,
      template[0].template[0].containers[3].name,
    ]
  }
}

// Grant the invoker SA (and any extra invokers) permission to execute jobs in each region.
resource "google_cloud_run_v2_job_iam_binding" "authorize-calls" {
  for_each = var.regions
  project  = var.project_id
  location = each.key
  name     = google_cloud_run_v2_job.this[each.key].name
  role     = "roles/run.invoker"
  members  = concat([google_service_account.invoker.member], var.invokers)
}

// One Cloud Scheduler job per region, each with its own schedule.
resource "google_cloud_scheduler_job" "this" {
  for_each = var.regions

  name             = "${var.name}-${each.key}"
  description      = "Triggers ${var.name} job in ${each.key}."
  schedule         = var.regional-cronspec[each.key].schedule
  time_zone        = var.regional-cronspec[each.key].time_zone
  paused           = var.regional-cronspec[each.key].paused
  attempt_deadline = "1800s"
  region           = each.key

  http_target {
    http_method = "POST"
    uri         = "https://${each.key}-run.googleapis.com/apis/run.googleapis.com/v1/namespaces/${var.project_id}/jobs/${google_cloud_run_v2_job.this[each.key].name}:run"

    // Cloud Run Jobs are invoked via the Google Admin API, which requires
    // an OAuth2 access token rather than an OIDC identity token.
    oauth_token {
      service_account_email = google_service_account.invoker.email
    }
  }
}
