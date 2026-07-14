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

  // The native histogram scrape keys need opentelemetry-collector-contrib
  // v0.142.0 or later; older collectors reject unknown keys at startup. When
  // scraping is disabled the rendered config omits the keys entirely, so those
  // collectors still start.
  // https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/receiver/prometheusreceiver/README.md#prometheus-native-histograms
  native_histograms_config = var.scrape_native_histograms ? join("\n", [
    "        # Negotiate the Prometheus protobuf format with targets that",
    "        # serve native histograms, which the text format cannot carry.",
    "        scrape_native_histograms: true",
    "        # Keep emitting classic _bucket series alongside native histograms",
    "        # so existing dashboards and recording rules keep working.",
    "        always_scrape_classic_histograms: true",
    "",
  ]) : ""

  // Cloud Run caps total CPU across all containers in a task at 8 vCPU
  // (8000m). Normalize each application container's cpu limit to millicpu
  // ("2" -> 2000, "1.5" -> 1500, "500m" -> 500) and add the otel sidecar's
  // implicit 1000m (it runs with no explicit limit) so an over-allocation
  // fails at plan time here instead of on a Cloud Run 400 at apply.
  container_millicpu = [
    for c in values(var.containers) : try(
      endswith(c.resources.limits.cpu, "m")
      ? tonumber(trimsuffix(c.resources.limits.cpu, "m"))
      : tonumber(c.resources.limits.cpu) * 1000,
      0
    )
    if try(c.resources.limits.cpu, null) != null
  ]
  otel_sidecar_millicpu = var.enable_otel_sidecar ? 1000 : 0
  total_millicpu        = sum(concat([0], local.container_millicpu, [local.otel_sidecar_millicpu]))
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

// The service account gets the three built-in observability roles
// individually, unless the caller passes observability_role: a single
// combined role (see the observability-role module) that costs one project
// IAM policy member entry instead of three, to keep projects with many
// services under the policy's 1,500-member limit.
resource "google_project_iam_member" "metrics-writer" {
  count = var.enable_observability_iam && var.observability_role == null ? 1 : 0

  project = var.project_id
  role    = "roles/monitoring.metricWriter"
  member  = "serviceAccount:${var.service_account}"
}

resource "google_project_iam_member" "trace-writer" {
  count = var.enable_observability_iam && var.observability_role == null ? 1 : 0

  project = var.project_id
  role    = "roles/cloudtrace.agent"
  member  = "serviceAccount:${var.service_account}"
}

resource "google_project_iam_member" "profiler-writer" {
  count = var.enable_observability_iam && var.observability_role == null ? 1 : 0

  project = var.project_id
  role    = "roles/cloudprofiler.agent"
  member  = "serviceAccount:${var.service_account}"
}

resource "google_project_iam_member" "observability" {
  count = var.enable_observability_iam && var.observability_role != null ? 1 : 0

  project = var.project_id
  role    = var.observability_role
  member  = "serviceAccount:${var.service_account}"
}

// State-address migration for deployments created when the three grants
// above had no count, so they see a rename rather than destroy/create.
moved {
  from = google_project_iam_member.metrics-writer
  to   = google_project_iam_member.metrics-writer[0]
}

moved {
  from = google_project_iam_member.trace-writer
  to   = google_project_iam_member.trace-writer[0]
}

moved {
  from = google_project_iam_member.profiler-writer
  to   = google_project_iam_member.profiler-writer[0]
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
            value = replace(replace(replace(replace(replace(file("${path.module}/otel-config/config.yaml"),
              "REPLACE_ME_TEAM", var.team),
              "REPLACE_ME_PROJECT_ID", var.project_id),
              "REPLACE_ME_NAME", var.name),
              "REPLACE_ME_TARGETS", local.metrics_targets),
            "        # REPLACE_ME_NATIVE_HISTOGRAMS\n", local.native_histograms_config)
          }
        }
      }

      dynamic "vpc_access" {
        for_each = each.value.network != null ? [1] : []
        content {
          network_interfaces {
            network    = each.value.network
            subnetwork = each.value.subnet
          }
          egress = var.egress
        }
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

    precondition {
      condition     = local.total_millicpu <= 8000
      error_message = "Cron ${var.name}: total CPU across all containers is ${local.total_millicpu}m, which exceeds the Cloud Run per-task limit of 8000m. The otel sidecar (enable_otel_sidecar=true) adds an implicit 1000m; lower the container cpu limit(s) to leave room for it."
    }
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

locals {
  success_alert_effective_duration = var.success_alert_duration_seconds > 0 ? var.success_alert_duration_seconds : var.success_alert_alignment_period_seconds
}

resource "google_monitoring_alert_policy" "success" {
  for_each = var.success_alert_alignment_period_seconds == 0 ? {} : var.regions

  alert_strategy {
    auto_close = "3600s"
  }

  display_name = "Cloud Run Job Success Execution: ${var.name} (${each.key})"
  combiner     = "OR"
  severity     = "ERROR"
  project      = var.project_id

  user_labels = local.merged_labels

  dynamic "documentation" {
    for_each = var.success_alert_documentation == "" ? [] : [var.success_alert_documentation]
    content {
      content   = documentation.value
      mime_type = "text/markdown"
    }
  }

  conditions {
    display_name = "Cloud Run Job Success Execution: ${var.name} (${each.key})"

    condition_absent {
      filter = <<EOT
        resource.type = "cloud_run_job"
        AND resource.labels.job_name = "${google_cloud_run_v2_job.this[each.key].name}"
        AND metric.type = "run.googleapis.com/job/completed_execution_count"
        AND metric.labels.result = "succeeded"
      EOT

      aggregations {
        alignment_period     = "${var.success_alert_alignment_period_seconds}s"
        cross_series_reducer = "REDUCE_NONE"
        per_series_aligner   = "ALIGN_MAX"
      }

      duration = var.success_alert_duration_seconds > 0 ? "${var.success_alert_duration_seconds}s" : "${var.success_alert_alignment_period_seconds}s"
      trigger {
        count = "1"
      }
    }
  }

  notification_channels = var.notification_channels
  enabled               = "true"
}
