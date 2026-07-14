// Create an identity as which the recorder service will run.
resource "google_service_account" "recorder" {
  project = var.project_id

  # This GSA doesn't need it's own audit rule because it is used in conjunction
  # with regional-go-service, which has a built-in audit rule.

  account_id   = var.name
  display_name = "Cloudevents recorder"
  description  = "Dedicated service account for our recorder service."

  lifecycle {
    # Fail fast rather than silently ignore dedicated-topic routing under the
    # gcs method, which subscribes via raw subscriptions this module does not
    # rewrite.
    precondition {
      condition     = (length(var.extra_brokers) == 0 && length(var.drop_shared_types) == 0) || local.use_custom_recorder
      error_message = "extra_brokers and drop_shared_types require method = \"trigger\"."
    }
    precondition {
      condition     = length(setsubtract(keys(var.extra_brokers), keys(var.types))) == 0
      error_message = "extra_brokers keys must be a subset of types (each needs a recorded BigQuery schema)."
    }
    precondition {
      condition     = length(setsubtract(var.drop_shared_types, keys(var.extra_brokers))) == 0
      error_message = "drop_shared_types must be a subset of extra_brokers keys, so a dropped type is still recorded from its dedicated topic."
    }
    precondition {
      # A dropped type must have an extra_brokers entry for every region, or the
      # recorder would lose coverage for that type in the uncovered regions.
      condition     = alltrue([for t in var.drop_shared_types : length(setsubtract(keys(var.regions), keys(lookup(var.extra_brokers, t, {})))) == 0])
      error_message = "each drop_shared_types entry must have an extra_brokers topic for every region in var.regions."
    }
  }
}

// The recorder service account is the only identity that should be writing
// to the regional GCS buckets.
resource "google_storage_bucket_iam_binding" "recorder-writes-to-gcs-buckets" {
  for_each = local.use_custom_recorder ? var.regions : {}

  bucket  = google_storage_bucket.recorder[each.key].name
  role    = "roles/storage.admin"
  members = ["serviceAccount:${google_service_account.recorder.email}"]
}

locals {
  lenv = [{
    name  = "LOG_PATH"
    value = "/logs"
  }]

  logrotate_env = var.flush_interval == "" ? local.lenv : concat(local.lenv, [{
    name  = "FLUSH_INTERVAL"
    value = var.flush_interval
  }])
}

module "this" {
  count              = local.use_custom_recorder ? 1 : 0
  source             = "../regional-go-service"
  observability_role = var.observability_role
  project_id         = var.project_id
  name               = var.name
  regions            = var.regions

  team                = var.team
  deletion_protection = var.deletion_protection
  service_account     = google_service_account.recorder.email
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
      resources = {
        limits = var.limits
      }
      regional-cpu-idle = var.cpu_idle
    }
    "logrotate" = {
      source = {
        working_dir = path.module
        importpath  = "./cmd/logrotate"
      }
      env = local.logrotate_env
      regional-env = [{
        name  = "BUCKET"
        value = { for k, v in google_storage_bucket.recorder : k => v.url }
      }]
      resources = {
        limits = {
          cpu    = "1000m"
          memory = "512Mi"
        }
      }
      volume_mounts = [{
        name       = "logs"
        mount_path = "/logs"
      }]
      regional-cpu-idle = var.cpu_idle
    }
  }
  volumes = [{
    name = "logs"
    empty_dir = {
      medium = var.local_disk_mount ? "DISK" : "MEMORY"
      size   = var.local_disk_mount ? "16G" : "2G"
    }
  }]

  scaling = var.scaling

  enable_profiler = var.enable_profiler

  notification_channels = var.notification_channels
}

resource "random_id" "trigger-suffix" {
  for_each    = var.types
  byte_length = 2
}

// Create a trigger for each region x type that sends events to the recorder
// service, skipping types dropped from the shared broker (recorded via
// extra_brokers instead).
module "triggers" {
  for_each = local.use_custom_recorder ? local.shared-regional-types : {}

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

  team = var.team

  notification_channels = var.notification_channels
}

// Additive triggers on dedicated topics for types routed off the shared broker.
// Distinct name suffix so they coexist with the shared triggers during cutover.
module "extra-triggers" {
  for_each = local.use_custom_recorder ? local.extra-regional-types : {}

  source = "../cloudevent-trigger"

  name       = "${var.name}-${random_id.trigger-suffix[each.value.type].hex}-x"
  project_id = var.project_id
  broker     = each.value.broker
  filter     = { "type" : each.value.type }

  depends_on = [module.this]
  private-service = {
    region = each.value.region
    name   = var.name
  }

  team = var.team

  notification_channels = var.notification_channels
}

module "recorder-dashboard" {
  source       = "../dashboard/cloudevent-receiver"
  service_name = var.name
  project_id   = var.project_id

  labels = { for type, schema in var.types : replace(type, ".", "_") => "" }

  notification_channels = var.notification_channels

  alerts = tomap({ for type, schema in var.types : "BQ DTS ${var.name}-${type}" => google_monitoring_alert_policy.bq_dts[type].id })

  split_triggers = var.split_triggers
  triggers = merge(
    // Shared-broker trigger per type, except types dropped from the shared
    // broker (their shared subscription no longer exists).
    {
      for type, schema in var.types : "type: ${type}" => {
        subscription_prefix   = "${var.name}-${random_id.trigger-suffix[type].hex}"
        alert_threshold       = schema.alert_threshold
        notification_channels = schema.notification_channels
      } if !contains(var.drop_shared_types, type)
    },
    // Dedicated-topic (extra) trigger per type routed off the shared broker.
    {
      for type in keys(var.extra_brokers) : "type: ${type} (dedicated)" => {
        subscription_prefix   = "${var.name}-${random_id.trigger-suffix[type].hex}-x"
        alert_threshold       = var.types[type].alert_threshold
        notification_channels = var.types[type].notification_channels
      }
    },
  )
}
