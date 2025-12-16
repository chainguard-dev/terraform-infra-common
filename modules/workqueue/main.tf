locals {
  sa_prefix = "${var.name}-"

  default_labels = {
    basename(abspath(path.module)) = var.name
    terraform-module               = basename(abspath(path.module))
  }

  squad_label = {
    squad = var.team
    team  = var.team
  }
  product_label = var.product != "" ? {
    product = var.product
  } : {}

  merged_labels = merge(local.default_labels, local.squad_label, local.product_label, var.labels)
}

resource "random_string" "bucket_suffix" {
  length  = 6 // Same length as "global"
  special = false
  upper   = false
  numeric = true
}

resource "google_storage_bucket" "global-workqueue" {
  name          = "${var.name}-${random_string.bucket_suffix.result}"
  project       = var.project_id
  location      = var.multi_regional_location
  force_destroy = true
  labels        = local.merged_labels

  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"
}

resource "google_storage_bucket_iam_binding" "global-authorize-access" {
  bucket = google_storage_bucket.global-workqueue.name
  role   = "roles/storage.admin"
  members = [
    "serviceAccount:${google_service_account.receiver.email}",
    "serviceAccount:${google_service_account.dispatcher.email}",
  ]
}

resource "google_pubsub_topic" "global-object-change-notifications" {
  for_each = var.regions

  name   = "${var.name}-global-${each.key}"
  labels = local.merged_labels

  message_storage_policy {
    allowed_persistence_regions = [each.key]
  }
}

data "google_storage_project_service_account" "gcs_account" {
  project = var.project_id
}

resource "google_pubsub_topic_iam_binding" "global-gcs-publishes-to-topic" {
  for_each = var.regions

  topic   = google_pubsub_topic.global-object-change-notifications[each.key].id
  role    = "roles/pubsub.publisher"
  members = ["serviceAccount:${data.google_storage_project_service_account.gcs_account.email_address}"]
}

resource "google_storage_notification" "global-object-change-notifications" {
  for_each = var.regions

  depends_on = [google_pubsub_topic_iam_binding.global-gcs-publishes-to-topic]

  bucket         = google_storage_bucket.global-workqueue.name
  payload_format = "JSON_API_V1"
  topic          = google_pubsub_topic.global-object-change-notifications[each.key].id
}
