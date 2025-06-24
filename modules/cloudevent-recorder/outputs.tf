output "dataset_id" {
  value = google_bigquery_dataset.this.dataset_id
}

output "table_ids" {
  value = {
    for k, v in google_bigquery_table.types : k => v.table_id
  }
}
