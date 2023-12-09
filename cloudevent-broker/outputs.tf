output "ingress" {
  depends_on  = [google_cloud_run_v2_service.this]
  description = "An object holding the name of the ingress service, which can be used to authorize callers to publish cloud events."
  value = {
    name = var.name
  }
}

output "broker" {
  depends_on  = [google_pubsub_topic.this]
  description = "A map from each of the input region names to the name of the Broker topic in each region.  These broker names are intended for use with the cloudevent-trigger module's broker input."
  value = {
    for region in keys(var.regions) : region => google_pubsub_topic.this[region].name
  }
}
