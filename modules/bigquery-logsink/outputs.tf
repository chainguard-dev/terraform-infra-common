output "dataset_id" {
  description = "The ID of the BigQuery dataset"
  value       = google_bigquery_dataset.this.dataset_id
}

output "table_ids" {
  description = "Map of table names to their IDs"
  value = {
    for k, v in google_bigquery_table.tables : k => v.table_id
  }
}

output "sink_names" {
  description = "Map of table names to their log sink names"
  value = {
    for k, v in google_logging_project_sink.sinks : k => v.name
  }
}
