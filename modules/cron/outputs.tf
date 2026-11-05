output "name" {
  value = module.impl.job_name
}

output "id" {
  value = module.impl.job_ids[var.region]
}
