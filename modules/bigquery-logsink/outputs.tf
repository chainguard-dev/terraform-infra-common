output "dataset_id" {
  description = "The ID of the BigQuery dataset"
  value       = google_bigquery_dataset.this.dataset_id
}

output "sink_names" {
  description = "Map of sink keys to their log sink names"
  value = {
    for k, v in google_logging_project_sink.sinks : k => v.name
  }
}

output "sink_writer_identities" {
  description = "Map of sink keys to their writer identity service accounts"
  value = {
    for k, v in google_logging_project_sink.sinks : k => v.writer_identity
  }
}
