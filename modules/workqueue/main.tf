locals {
  sa_prefix = "${var.name}-"

  default_labels = {
    basename(abspath(path.module)) = var.name
    terraform-module               = basename(abspath(path.module))
  }

  squad_label = var.squad != "" ? {
    squad = var.squad
    team  = var.squad
  } : {}
  product_label = var.product != "" ? {
    product = var.product
  } : {}

  merged_labels = merge(local.default_labels, local.squad_label, local.product_label, var.labels)
}

// Storage buckets: regional and global
resource "google_storage_bucket" "workqueue" {
  for_each = var.regions

  name          = "${var.name}-${each.key}"
  project       = var.project_id
  location      = each.key
  force_destroy = true
  labels        = local.merged_labels

  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"
}

resource "google_storage_bucket" "global-workqueue" {
  name          = "${var.name}-global"
  project       = var.project_id
  location      = var.multi_regional_location
  force_destroy = true
  labels        = local.merged_labels

  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"
}

// Storage IAM: regional and global
resource "google_storage_bucket_iam_binding" "authorize-access" {
  for_each = var.regions

  bucket = google_storage_bucket.workqueue[each.key].name
  role   = "roles/storage.admin"
  members = [
    "serviceAccount:${google_service_account.receiver.email}",
    "serviceAccount:${google_service_account.dispatcher.email}",
  ]
}

resource "google_storage_bucket_iam_binding" "global-authorize-access" {
  bucket = google_storage_bucket.global-workqueue.name
  role   = "roles/storage.admin"
  members = [
    "serviceAccount:${google_service_account.receiver.email}",
    "serviceAccount:${google_service_account.dispatcher.email}",
  ]
}

// Pub/Sub topics: regional and global
resource "google_pubsub_topic" "object-change-notifications" {
  for_each = var.regions
  name     = "${var.name}-${each.key}"
  labels   = local.merged_labels

  message_storage_policy {
    allowed_persistence_regions = [each.key]
  }
}

resource "google_pubsub_topic" "global-object-change-notifications" {
  for_each = var.regions

  name   = "${var.name}-global-${each.key}"
  labels = local.merged_labels

  message_storage_policy {
    allowed_persistence_regions = [each.key]
  }
}

data "google_storage_project_service_account" "gcs_account" {}

// Pub/Sub IAM: regional and global
resource "google_pubsub_topic_iam_binding" "gcs-publishes-to-topic" {
  for_each = var.regions

  topic   = google_pubsub_topic.object-change-notifications[each.key].id
  role    = "roles/pubsub.publisher"
  members = ["serviceAccount:${data.google_storage_project_service_account.gcs_account.email_address}"]
}

resource "google_pubsub_topic_iam_binding" "global-gcs-publishes-to-topic" {
  for_each = var.regions

  topic   = google_pubsub_topic.global-object-change-notifications[each.key].id
  role    = "roles/pubsub.publisher"
  members = ["serviceAccount:${data.google_storage_project_service_account.gcs_account.email_address}"]
}

// Storage notifications: regional and global
resource "google_storage_notification" "object-change-notifications" {
  for_each = var.regions

  depends_on = [google_pubsub_topic_iam_binding.gcs-publishes-to-topic]

  bucket         = google_storage_bucket.workqueue[each.key].name
  payload_format = "JSON_API_V1"
  topic          = google_pubsub_topic.object-change-notifications[each.key].id
}

resource "google_storage_notification" "global-object-change-notifications" {
  for_each = var.regions

  depends_on = [google_pubsub_topic_iam_binding.global-gcs-publishes-to-topic]

  bucket         = google_storage_bucket.global-workqueue.name
  payload_format = "JSON_API_V1"
  topic          = google_pubsub_topic.global-object-change-notifications[each.key].id
}
