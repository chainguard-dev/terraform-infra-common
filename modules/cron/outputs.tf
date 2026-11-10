output "name" {
  value = module.impl.job_name
}

output "id" {
  value = module.impl.job_ids[var.region]
}

output "invocation_grants" {
  description = "Job invocation bindings managed by this module, keyed by region. Excludes inherited and out-of-band IAM grants."
  value       = module.impl.invocation_grants
}

output "tag_bindings" {
  description = "Cloud Run tag binding parents, values and locations, keyed by region/tag key."
  value       = module.impl.tag_bindings
}

output "product_env" {
  description = "Rendered CHAINGUARD_PRODUCT values from the job containers, keyed by region."
  sensitive   = true
  value = { for region, containers in module.impl.containers : region => toset(flatten([
    for container in containers : [for env in container.env : env.value if env.name == "CHAINGUARD_PRODUCT"]
  ])) }
}
