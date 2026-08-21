output "cluster_name" {
  value = google_container_cluster.this.name
}

output "cluster_id" {
  value = google_container_cluster.this.id
}

output "service_account_email" {
  value = google_service_account.cluster_default.email
}

output "cluster_endpoint" {
  value     = google_container_cluster.this.endpoint
  sensitive = true
}

output "cluster_ca_certificate" {
  # one() instead of [0]: master_auth is computed and empty in mocked test
  # plans; on a real cluster it always has exactly one element.
  value     = one(google_container_cluster.this.master_auth[*].cluster_ca_certificate)
  sensitive = true
}
