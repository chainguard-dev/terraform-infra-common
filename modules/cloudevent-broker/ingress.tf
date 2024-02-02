// Create a dedicated identity as which to run the broker ingress service
// (and authorize it's actions)
resource "google_service_account" "this" {
  project = var.project_id

  account_id   = var.name
  display_name = "Broker Ingress"
  description  = "A dedicated identity for the ${var.name} broker ingress to operate as."
}

// Authorize the ingress identity to publish events to each of
// the regional broker topics.
// NOTE: we use binding vs. member because we do not expect anything
// to publish to this topic other than the ingress service.
resource "google_pubsub_topic_iam_binding" "ingress-publishes-events" {
  for_each = var.regions

  project = var.project_id
  topic   = google_pubsub_topic.this[each.key].name
  role    = "roles/pubsub.publisher"
  members = ["serviceAccount:${google_service_account.this.email}"]
}

module "this" {
  source     = "../regional-go-service"
  project_id = var.project_id
  name       = var.name
  regions    = var.regions

  service_account = google_service_account.this.email
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
    }
  }
}

module "topic" {
  source       = "../dashboard/sections/topic"
  title        = "Broker Events"
  topic_prefix = var.name
}

module "logs" {
  source = "../dashboard/sections/logs"
  title  = "Service Logs"
  filter = ["resource.type=\"cloud_run_revision\""]
}

module "http" {
  source       = "../dashboard/sections/http"
  title        = "HTTP"
  filter       = ["resource.type=\"cloud_run_revision\""]
  service_name = var.name
}

module "resources" {
  source        = "../dashboard/sections/resources"
  title         = "Resources"
  filter        = ["resource.type=\"cloud_run_revision\"", "resource.labels.service_name=\"${var.name}\""]
  cloudrun_name = var.name
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

resource "google_monitoring_dashboard" "dashboard" {
  dashboard_json = jsonencode({
    displayName = "Cloud Events Broker Ingress: ${var.name}"
    labels = {
      "service" : ""
      "eventing" : ""
    }
    dashboardFilters = [{
      filterType  = "RESOURCE_LABEL"
      stringValue = var.name
      labelKey    = "service_name"
    }]

    // https://cloud.google.com/monitoring/api/ref_v3/rest/v1/projects.dashboards#mosaiclayout
    mosaicLayout = {
      columns = module.width.size
      tiles   = module.layout.tiles,
    }
  })
}
