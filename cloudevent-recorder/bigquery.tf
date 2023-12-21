// The BigQuery dataset that will hold the recorded cloudevents.
resource "google_bigquery_dataset" "this" {
  project    = var.project_id
  dataset_id = "cloudevents_${replace(var.name, "-", "_")}"
  location   = var.location

  default_partition_expiration_ms = (var.retention-period) * 24 * 60 * 60 * 1000
}

// A BigQuery table for each of the cloudevent types with the specified
// schema for that type.
resource "google_bigquery_table" "types" {
  for_each = var.types

  project    = var.project_id
  dataset_id = google_bigquery_dataset.this.dataset_id
  table_id   = replace(each.key, ".", "_")
  schema     = each.value

  require_partition_filter = false

  time_partitioning {
    type = "DAY"

    expiration_ms = (var.retention-period) * 24 * 60 * 60 * 1000
  }

  deletion_protection = var.deletion_protection
}

// Create an identity that will be used to run the BQ DTS job,
// which we will grant the necessary permissions to.
resource "google_service_account" "import-identity" {
  project      = var.project_id
  account_id   = "${var.name}-import"
  display_name = "BigQuery import identity"
}

// Grant the import identity permission to manipulate the dataset's tables.
resource "google_bigquery_table_iam_member" "import-writes-to-tables" {
  for_each = var.types

  project    = var.project_id
  dataset_id = google_bigquery_dataset.this.dataset_id
  table_id   = google_bigquery_table.types[each.key].table_id
  role       = "roles/bigquery.admin"
  member     = "serviceAccount:${google_service_account.import-identity.email}"
}

// Grant the import identity permission to read the event data from
// the regional GCS buckets.
resource "google_storage_bucket_iam_member" "import-reads-from-gcs-buckets" {
  for_each = var.regions

  bucket = google_storage_bucket.recorder[each.key].name
  role   = "roles/storage.objectViewer"
  member = "serviceAccount:${google_service_account.import-identity.email}"
}

// Grant the BQ DTS service account for this project permission to assume
// the identity we are assigning to the DTS job.
resource "google_service_account_iam_member" "bq-dts-assumes-import-identity" {
  service_account_id = google_service_account.import-identity.name
  role               = "roles/iam.serviceAccountShortTermTokenMinter"
  member             = "serviceAccount:service-${data.google_project.project.number}@gcp-sa-bigquerydatatransfer.iam.gserviceaccount.com"
}

// Only users that can "act as" the service account can set the service account on a transfer job.
resource "google_service_account_iam_member" "provisioner-acts-as-import-identity" {
  service_account_id = google_service_account.import-identity.name
  role               = "roles/iam.serviceAccountUser"
  member             = var.provisioner
}

// Create a BQ DTS job for each of the regions x types pulling from the appropriate buckets and paths.
resource "google_bigquery_data_transfer_config" "import-job" {
  for_each = local.regional-types

  depends_on = [google_service_account_iam_member.provisioner-acts-as-import-identity]

  project              = var.project_id
  display_name         = "${var.name}-${each.key}"
  location             = google_bigquery_dataset.this.location // These must be colocated
  service_account_name = google_service_account.import-identity.email
  disabled             = false

  data_source_id         = "google_cloud_storage"
  schedule               = "every 15 minutes"
  destination_dataset_id = google_bigquery_dataset.this.dataset_id

  // TODO(mattmoor): Bring back pubsub notification.
  # notification_pubsub_topic = google_pubsub_topic.bq_notification[each.key].id
  params = {
    data_path_template              = "gs://${google_storage_bucket.recorder[each.value.region].name}/${each.value.type}/*"
    destination_table_name_template = google_bigquery_table.types[each.value.type].table_id
    file_format                     = "JSON"
    max_bad_records                 = 0
    delete_source_files             = false
  }
}
