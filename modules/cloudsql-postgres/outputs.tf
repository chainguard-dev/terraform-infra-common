output "instance_name" {
  description = "Name of the primary Cloud SQL instance."
  value       = google_sql_database_instance.this.name
}

output "instance_connection_name" {
  description = "Fully‑qualified connection name of the primary instance (<project>:<region>:<instance>)."
  value       = google_sql_database_instance.this.connection_name
}

output "private_ip_address" {
  description = "Private IPv4 address of the primary Cloud SQL instance."
  value       = google_sql_database_instance.this.private_ip_address
}

output "instance_self_link" {
  description = "Self‑link URI of the primary Cloud SQL instance."
  value       = google_sql_database_instance.this.self_link
}

output "replica_connection_names" {
  description = "Map of replica region → connection name. Empty if no replicas."
  value       = { for r, inst in google_sql_database_instance.replicas : r => inst.connection_name }
}

output "replica_private_ips" {
  description = "Map of replica region → private IPv4 address. Empty if no replicas."
  value       = { for r, inst in google_sql_database_instance.replicas : r => inst.private_ip_address }
}

output "client_sa_bindings" {
  description = "Map of service‑account email → IAM binding resource ID."
  value       = { for k, v in google_project_iam_member.client_sa : k => v.id }
}

# Private Service Connect (PSC) outputs

output "psc_service_attachment_link" {
  description = "The PSC service attachment link for connecting from consumer projects. Only populated when PSC is enabled."
  value       = var.psc_enabled ? google_sql_database_instance.this.psc_service_attachment_link : null
}

output "psc_dns_name" {
  description = "The DNS name to use for PSC connections. Only populated when PSC is enabled."
  value       = var.psc_enabled ? google_sql_database_instance.this.dns_name : null
}
