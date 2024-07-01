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
