output "names" {
  value = {
    for k, v in google_cloud_run_v2_service.this : k => v.name
  }
}

output "locations" {
  value = {
    for k, v in google_cloud_run_v2_service.this : k => v.location
  }
}

output "uris" {
  value = {
    for k, v in google_cloud_run_v2_service.this : k => v.uri
  }
}

output "access_policy" {
  description = "Regional ingress, VPC egress, and invocation policy managed by this module; excludes out-of-band IAM grants."
  value = {
    for region, service in google_cloud_run_v2_service.this : region => {
      ingress = service.ingress
      egress  = service.template[0].vpc_access[0].egress

      public_invoker = contains(keys(google_cloud_run_v2_service_iam_member.public-services-are-unauthenticated), region)
    }
  }
}

output "containers" {
  description = "Rendered regional container configuration, including environment and secret references."
  sensitive   = true
  value       = { for region, service in google_cloud_run_v2_service.this : region => service.template[0].containers }
}
