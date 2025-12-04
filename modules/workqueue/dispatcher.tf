locals {
  dispatcher_batch_size = var.batch-size != null ? var.batch-size : ceil(var.concurrent-work / length(var.regions))
}

// Compute a suffix that satisfies the regex:
// ^[a-z](?:[-a-z0-9]{4,28}[a-z0-9])$
resource "random_string" "dispatcher" {
  length  = 30 - length(local.sa_prefix)
  special = false
  upper   = false
}

// Create a dedicated GSA for the dispatcher service.
resource "google_service_account" "dispatcher" {
  project = var.project_id

  account_id   = "${local.sa_prefix}${random_string.dispatcher.result}"
  display_name = "Workqueue Dispatcher"
  description  = "The identity as which the workqueue dispatcher service runs for the ${var.name} workqueue."
}

// Authorize the dispatcher service account to call the target.
module "dispatcher-calls-target" {
  for_each = var.regions

  source = "../authorize-private-service"

  project_id = var.project_id
  region     = each.key
  name       = var.reconciler-service.name

  service-account = google_service_account.dispatcher.email
}

// Stand up the dispatcher service in each of our regions.
module "dispatcher-service" {
  source     = "../regional-go-service"
  project_id = var.project_id
  name       = "${var.name}-dsp"
  regions    = var.regions
  labels     = merge({ "service" : "workqueue-dispatcher" }, local.default_labels)
  team       = var.team
  product    = var.product

  # Give the things in the workqueue a lot of time to process the key.
  request_timeout_seconds = 3600

  deletion_protection = var.deletion_protection

  service_account = google_service_account.dispatcher.email
  containers = {
    "dispatcher" = {
      source = {
        working_dir = path.module
        importpath  = "github.com/chainguard-dev/terraform-infra-common/modules/workqueue/cmd/dispatcher"
      }
      ports = [{
        name           = "h2c"
        container_port = 8080
      }]
      env = [
        {
          name  = "WORKQUEUE_MODE"
          value = "gcs"
        },
        {
          name  = "WORKQUEUE_CONCURRENCY"
          value = "${var.concurrent-work}"
        },
        {
          name  = "WORKQUEUE_MAX_RETRY"
          value = "${var.max-retry}"
        },
        {
          name  = "WORKQUEUE_BATCH_SIZE"
          value = tostring(local.dispatcher_batch_size)
        }
      ]
      regional-env = [
        {
          name = "WORKQUEUE_BUCKET"
          value = {
            for k, v in var.regions : k => google_storage_bucket.global-workqueue.name
          }
        },
        {
          name  = "WORKQUEUE_TARGET"
          value = { for k, v in module.dispatcher-calls-target : k => v.uri }
        },
      ]
      regional-cpu-idle = lookup(var.cpu_idle, "dispatcher", {})
    }
  }

  notification_channels = var.notification_channels
}

// Compute a suffix that satisfies the regex:
// ^[a-z](?:[-a-z0-9]{4,28}[a-z0-9])$
resource "random_string" "cron-trigger" {
  length  = 30 - length(local.sa_prefix)
  special = false
  upper   = false
}

// Create a dedicated GSA for the cron trigger.
resource "google_service_account" "cron-trigger" {
  project = var.project_id

  account_id   = "${local.sa_prefix}${random_string.cron-trigger.result}"
  display_name = "Workqueue Cron Trigger"
  description  = "The identity as which the cloud scheduler will invoke the ${var.name} dispatcher."
}

// Authorize the cron-trigger service account to call the dispatcher.
module "cron-trigger-calls-dispatcher" {
  for_each = var.regions

  source = "../authorize-private-service"

  depends_on = [module.dispatcher-service]

  project_id = var.project_id
  region     = each.key
  name       = "${var.name}-dsp"

  service-account = google_service_account.cron-trigger.email
}

resource "google_cloud_scheduler_job" "cron" {
  for_each = var.regions

  name        = "${var.name}-${each.key}"
  description = "Periodically trigger the dispatcher to dispatch work."
  // Schedule this to run every minute.  We do this more frequently now
  // because otherwise we risk delaying tasks with a NotBefore for up to 30m
  // if the workqueue is otherwise idle.
  schedule         = "* * * * *"
  time_zone        = "America/New_York"
  attempt_deadline = "1800s" // The maximum
  region           = each.key

  http_target {
    http_method = "GET"
    uri         = module.cron-trigger-calls-dispatcher[each.key].uri

    oidc_token {
      service_account_email = google_service_account.cron-trigger.email
      // There is a provider bug, so despite this being the default, we provide it explicitly.
      audience = module.cron-trigger-calls-dispatcher[each.key].uri
    }
  }
}

// Compute a suffix that satisfies the regex:
// ^[a-z](?:[-a-z0-9]{4,28}[a-z0-9])$
resource "random_string" "change-trigger" {
  length  = 30 - length(local.sa_prefix)
  special = false
  upper   = false
}

// Create a dedicated GSA for the object change notification subscription.
resource "google_service_account" "change-trigger" {
  project = var.project_id

  account_id   = "${local.sa_prefix}${random_string.change-trigger.result}"
  display_name = "Workqueue Change Trigger"
  description  = "The identity as which the pubsub object change subscription will invoke the ${var.name} dispatcher."
}

// Lookup the identity of the pubsub service agent.
resource "google_project_service_identity" "pubsub" {
  provider = google-beta
  project  = var.project_id
  service  = "pubsub.googleapis.com"
}

// Authorize Pub/Sub to impersonate the delivery service account to authorize
// deliveries using this service account.
// NOTE: we use binding vs. member because we expect nothing but pubsub to be
// able to assume this identity.
resource "google_service_account_iam_binding" "allow-pubsub-to-mint-tokens" {
  service_account_id = google_service_account.change-trigger.name

  role    = "roles/iam.serviceAccountTokenCreator"
  members = ["serviceAccount:${google_project_service_identity.pubsub.email}"]
}

// Authorize the change-trigger service account to call the dispatcher.
module "change-trigger-calls-dispatcher" {
  for_each = var.regions

  source = "../authorize-private-service"

  depends_on = [module.dispatcher-service]

  project_id = var.project_id
  region     = each.key
  name       = "${var.name}-dsp"

  service-account = google_service_account.change-trigger.email
}

resource "google_pubsub_subscription" "global-this" {
  for_each = var.regions

  name   = "${var.name}-global-${each.key}"
  topic  = google_pubsub_topic.global-object-change-notifications[each.key].id
  labels = merge({ "service" : "workqueue-dispatcher" }, local.merged_labels)

  ack_deadline_seconds = 600 // Maximum value

  push_config {
    push_endpoint = module.change-trigger-calls-dispatcher[each.key].uri

    // Authenticate requests to this service using tokens minted
    // from the given service account.
    oidc_token {
      service_account_email = google_service_account.change-trigger.email
    }
  }

  expiration_policy {
    ttl = "" // This does not expire.
  }
}
