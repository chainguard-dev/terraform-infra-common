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
  value     = google_container_cluster.this.master_auth[0].cluster_ca_certificate
  sensitive = true
}

output "cluster_pod_ipv4_cidr_block" {
  description = "The cluster's pod secondary IPv4 CIDR range (ip_allocation_policy.cluster_ipv4_cidr_block)."
  value       = google_container_cluster.this.ip_allocation_policy[0].cluster_ipv4_cidr_block
}
