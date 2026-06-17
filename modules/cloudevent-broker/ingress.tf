// Create a dedicated identity as which to run the broker ingress service
// (and authorize it's actions)
resource "google_service_account" "this" {
  project = var.project_id

  # This GSA doesn't need it's own audit rule because it is used in conjunction
  # with regional-go-service, which has a built-in audit rule.

  account_id   = var.name
  display_name = "Broker Ingress"
  description  = "A dedicated identity for the ${var.name} broker ingress to operate as."
}

// Authorize the ingress identity (and any extra_publishers) to publish
// events to each of the regional broker topics.
// NOTE: we use binding vs. member so this list is authoritative — any
// additional publishers must be declared here via extra_publishers rather
// than through separate google_pubsub_topic_iam_member resources, which
// would be wiped on every apply.
resource "google_pubsub_topic_iam_binding" "ingress-publishes-events" {
  for_each = var.regions

  project = var.project_id
  topic   = google_pubsub_topic.this[each.key].name
  role    = "roles/pubsub.publisher"
  members = concat(
    ["serviceAccount:${google_service_account.this.email}"],
    [for sa in var.extra_publishers : "serviceAccount:${sa}"],
  )
}

module "this" {
  source     = "../regional-go-service"
  project_id = var.project_id
  name       = var.name
  regions    = var.regions
  team       = var.team

  deletion_protection               = var.deletion_protection
  ingress                           = var.ingress
  require_authenticated_invocations = var.require_authenticated_invocations
  service_account                   = google_service_account.this.email
  containers = {
    "ingress" = {
      source = {
        working_dir = path.module
        importpath  = "./cmd/ingress"
      }
      ports = [{ container_port = 8080 }]
      regional-env = [{
        name  = "PUBSUB_TOPIC"
        value = { for k, v in google_pubsub_topic.this : k => v.name }
      }]
      resources = {
        limits = var.limits
      }
      regional-cpu-idle = var.cpu_idle
    }
  }

  scaling = var.scaling

  enable_profiler = var.enable_profiler

  notification_channels = var.notification_channels
}

module "topic" {
  source       = "../dashboard/sections/topic"
  title        = "Broker Events"
  topic_prefix = var.name
}

module "logs" {
  source        = "../dashboard/sections/logs"
  title         = "Service Logs"
  filter        = []
  cloudrun_type = "service"
}

module "http" {
  source       = "../dashboard/sections/http"
  title        = "HTTP"
  filter       = []
  service_name = var.name
}

module "resources" {
  source        = "../dashboard/sections/resources"
  title         = "Resources"
  filter        = []
  cloudrun_name = var.name
  cloudrun_type = "service"

  notification_channels = var.notification_channels
}

module "width" { source = "../dashboard/sections/width" }

module "layout" {
  source = "../dashboard/sections/layout"
  sections = [
    module.topic.section,
    module.logs.section,
    module.http.section,
    module.resources.section,
  ]
}

module "dashboard" {
  source = "../dashboard"

  object = {
    displayName = "Cloud Events Broker Ingress: ${var.name}"
    labels = {
      "service" : ""
      "eventing" : ""
    }
    dashboardFilters = [
      {
        # for GCP Cloud Run built-in metrics
        filterType  = "RESOURCE_LABEL"
        stringValue = var.name
        labelKey    = "service_name"
      },
      {
        # for Prometheus user added metrics
        filterType  = "METRIC_LABEL"
        stringValue = var.name
        labelKey    = "service_name"
      },
    ]

    // https://cloud.google.com/monitoring/api/ref_v3/rest/v1/projects.dashboards#mosaiclayout
    mosaicLayout = {
      columns = module.width.size
      tiles   = module.layout.tiles,
    }
  }
}
