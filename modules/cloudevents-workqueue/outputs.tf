output "subscriber" {
  value = {
    uris            = module.subscriber.uris
    names           = module.subscriber.names
    locations       = module.subscriber.locations
    service_account = google_service_account.subscriber.email
  }
}
