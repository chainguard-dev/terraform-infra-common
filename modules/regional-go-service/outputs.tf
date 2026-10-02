output "names" {
  value = module.this.names
}

output "locations" {
  value = module.this.locations
}

output "uris" {
  value = module.this.uris
}

output "access_policy" {
  description = "Regional ingress, VPC egress, and invocation policy managed by this module; excludes out-of-band IAM grants."
  value       = module.this.access_policy
}

output "containers" {
  description = "Rendered regional container configuration, including environment and secret references."
  sensitive   = true
  value       = module.this.containers
}
