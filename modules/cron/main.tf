terraform {
  required_providers {
    google = { source = "hashicorp/google" }
    // Required transitively by the regional-go-cron child module; declared here
    // so provider configurations attach at the root module.
    google-beta = { source = "hashicorp/google-beta" }
  }
}

resource "google_project_service" "cloud_run_api" {
  project = var.project_id
  service = "run.googleapis.com"

  disable_on_destroy = false
}

resource "google_project_service" "cloudscheduler" {
  project = var.project_id
  service = "cloudscheduler.googleapis.com"

  disable_on_destroy = false
}

module "impl" {
  source = "../regional-go-cron"

  project_id      = var.project_id
  name            = "${var.name}-cron"
  service_account = var.service_account
  team            = var.team
  product         = var.product

  resource_manager_tags = var.resource_manager_tags

  enable_observability_iam = var.enable_observability_iam
  egress                   = var.vpc_access != null ? var.vpc_access.egress : null

  regions = { (var.region) = var.vpc_access != null ? {
    network = var.vpc_access.network_interfaces[0].network
    subnet  = var.vpc_access.network_interfaces[0].subnetwork
  } : {} }

  regional-cronspec = { (var.region) = {
    schedule  = var.schedule
    time_zone = "UTC"
    paused    = var.paused
  } }

  containers = {
    "this" = {
      source = {
        base_image  = var.base_image
        working_dir = var.working_dir
        importpath  = var.importpath
        env         = var.ko_build_env
      }
      command   = var.command
      args      = var.args
      resources = { limits = { cpu = var.cpu, memory = var.memory } }
      env = concat(
        [for k, v in var.env : { name = k, value = v }],
        [for k, v in var.secret_env : { name = k, value = null, value_source = { secret_key_ref = { secret = split("@", v)[0], version = try(split("@", v)[1], "latest") } } }],
        [{ name = "CHAINGUARD_TEAM", value = var.team }],
        [{ name = "CHAINGUARD_PRODUCT", value = var.product }],
      )
      volume_mounts = var.volume_mounts
    }
  }

  volumes                  = var.volumes
  max_retries              = var.max_retries
  timeout                  = var.timeout
  task_count               = var.task_count
  parallelism              = var.parallelism
  execution_environment    = var.execution_environment
  launch_stage             = var.launch_stage
  deletion_protection      = var.deletion_protection
  notification_channels    = var.notification_channels
  labels                   = var.labels
  enable_otel_sidecar      = var.enable_otel_sidecar
  otel_collector_image     = var.otel_collector_image
  scrape_native_histograms = var.scrape_native_histograms
  observability_role       = var.observability_role
  invokers                 = var.invokers

  success_alert_alignment_period_seconds = var.success_alert_alignment_period_seconds
  success_alert_duration_seconds         = var.success_alert_duration_seconds
  success_alert_documentation            = var.success_alert_documentation
  failed_execution_alert                 = var.failed_execution_alert
}


// Call cloud run api to execute job once.
// https://cloud.google.com/run/docs/execute/jobs#command-line
resource "null_resource" "exec" {
  count = var.exec ? 1 : 0

  provisioner "local-exec" {
    // --async is load-bearing, not a tidier spelling of "no --wait". gcloud
    // has three modes here: --wait blocks until the execution COMPLETES,
    // --async returns as soon as it is CREATED, and the default -- neither
    // flag -- blocks until it starts RUNNING, which means provisioning a task
    // and pulling a freshly pushed image. That default is what exec_wait=false
    // used to select, and it is not what this variable promises: dev-eco's
    // role-init took 5m43s to print "has successfully started running" on an
    // apply that had opted out of waiting.
    command = join(" ", concat([
      "gcloud",
      "--project=${var.project_id}",
      "run",
      "jobs",
      "execute",
      module.impl.job_name,
      "--region=${var.region}",
    ], var.exec_wait ? ["--wait"] : ["--async"]))
  }

  triggers = {
    // Re-run the exec provisioner whenever the container images change.
    // image_refs are computed by ko/cosign before the Cloud Run Job is updated,
    // so these values are stable at apply time (unlike job_etag or job_generation,
    // which are server-side computed values that change mid-apply).
    image_refs = join(",", values(module.impl.image_refs))
  }
}

moved {
  from = google_cloud_run_v2_job.job
  to   = module.impl.google_cloud_run_v2_job.this["us-central1"]
}
moved {
  from = google_cloud_scheduler_job.cron
  to   = module.impl.google_cloud_scheduler_job.this["us-central1"]
}
moved {
  from = google_project_iam_member.metrics-writer
  to   = module.impl.google_project_iam_member.metrics-writer
}
moved {
  from = google_project_iam_member.trace-writer
  to   = module.impl.google_project_iam_member.trace-writer
}
moved {
  from = google_project_iam_member.profiler-writer
  to   = module.impl.google_project_iam_member.profiler-writer
}
moved {
  from = google_monitoring_alert_policy.success[0]
  to   = module.impl.google_monitoring_alert_policy.success["us-central1"]
}
