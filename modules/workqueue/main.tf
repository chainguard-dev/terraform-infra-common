locals {
  sa_prefix = "${var.name}-"
}

resource "google_storage_bucket" "workqueue" {
  for_each = var.regions

  name          = "${var.name}-${each.key}"
  project       = var.project_id
  location      = each.key
  force_destroy = true

  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"
}

resource "google_storage_bucket_iam_binding" "authorize-access" {
  for_each = var.regions

  bucket = google_storage_bucket.workqueue[each.key].name
  role = "roles/storage.admin"
  members = [
    "serviceAccount:${google_service_account.receiver.email}",
    "serviceAccount:${google_service_account.dispatcher.email}",
  ]
}

// Create a topic per region for the regional buckets to route events to.
resource "google_pubsub_topic" "object-change-notifications" {
  for_each = var.regions
  name     = "${var.name}-${each.key}"

  message_storage_policy {
    allowed_persistence_regions = [each.key]
  }
}

data "google_storage_project_service_account" "gcs_account" {}

resource "google_pubsub_topic_iam_binding" "gcs-publishes-to-topic" {
  for_each = var.regions

  topic   = google_pubsub_topic.object-change-notifications[each.key].id
  role    = "roles/pubsub.publisher"
  members = ["serviceAccount:${data.google_storage_project_service_account.gcs_account.email_address}"]
}

resource "google_storage_notification" "object-change-notifications" {
  for_each = var.regions

  depends_on = [ google_pubsub_topic_iam_binding.gcs-publishes-to-topic ]

  bucket         = google_storage_bucket.workqueue[each.key].name
  payload_format = "JSON_API_V1"
  topic          = google_pubsub_topic.object-change-notifications[each.key].id
}
