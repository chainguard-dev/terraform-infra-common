output "dataset_id" {
  value = var.create_dataset ? google_bigquery_dataset.this[0].dataset_id : data.google_bigquery_dataset.existing[0].dataset_id
}

output "table_ids" {
  value = {
    for k, v in google_bigquery_table.types : k => v.table_id
  }
}
