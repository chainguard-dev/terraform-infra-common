resource "google_logging_project_sink" "sinks" {
  for_each = var.tables

  project = var.project_id
  name    = "${var.name}_${each.key}_sink"

  # Route logs to the BigQuery table
  destination = "bigquery.googleapis.com/projects/${var.project_id}/datasets/${google_bigquery_dataset.this.dataset_id}/tables/${google_bigquery_table.tables[each.key].table_id}"

  # Apply the log filter for this table
  filter = each.value.log_filter

  # Use partitioned tables if enabled
  bigquery_options {
    use_partitioned_tables = var.use_partitioned_tables
  }

  # Ensures sink is created/updated after the table exists
  depends_on = [google_bigquery_table.tables]
}

# Grant the sink's writer identity permission to write to BigQuery
resource "google_bigquery_dataset_iam_member" "sink_writers" {
  for_each = var.tables

  project    = var.project_id
  dataset_id = google_bigquery_dataset.this.dataset_id
  role       = "roles/bigquery.dataEditor"
  member     = google_logging_project_sink.sinks[each.key].writer_identity
}
