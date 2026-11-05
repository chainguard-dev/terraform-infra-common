output "name" {
  value = module.impl.job_name
}

output "id" {
  value = module.impl.job_ids[var.region]
}

output "invocation_grants" {
  description = "Job invocation bindings managed by this module, keyed by region; excludes inherited and out-of-band IAM grants."
  value       = module.impl.invocation_grants
}
