resource "google_bigquery_table" "tables" {
  for_each = var.tables

  project    = var.project_id
  dataset_id = google_bigquery_dataset.this.dataset_id
  table_id   = each.key

  description = each.value.description

  deletion_protection = var.deletion_protection

  # Parse and set the schema
  schema = each.value.schema

  # Time partitioning configuration (required)
  time_partitioning {
    type                     = "DAY"
    field                    = each.value.partition_field
    expiration_ms            = var.partition_expiration_days * 24 * 60 * 60 * 1000
    require_partition_filter = false
  }

  clustering = each.value.clustering_fields

  labels = local.merged_labels
}
