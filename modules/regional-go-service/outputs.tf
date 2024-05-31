output "names" {
  value = module.this.names
}

output "uris" {
  value = {
    for k, v in google_cloud_run_v2_service.this : k => v.uri
  }
}
