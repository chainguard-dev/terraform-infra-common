output "names" {
  value = module.this.names
}

output "name" {
  description = "The service name, which every region shares. Read off the services, so a resource that names the service is created after it."
  value       = module.this.name
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

output "network_tags" {
  description = "Network tags on each region's direct VPC egress interface, keyed by region. An untagged region and a region in regional-connector report null."
  value       = module.this.network_tags
}

output "containers" {
  description = "Rendered regional container configuration, including environment and secret references."
  sensitive   = true
  value       = module.this.containers
}

output "tag_bindings" {
  description = "Cloud Run tag binding parents, values and locations, keyed by region/tag key."
  value       = module.this.tag_bindings
}
