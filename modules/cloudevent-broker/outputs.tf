output "ingress" {
  depends_on  = [module.this]
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

output "dedicated" {
  depends_on  = [google_pubsub_topic.dedicated]
  description = "A map from each dedicated ce-type to a per-region map of its dedicated topic name. Empty unless dedicated_topics is set. Intended for use as the broker input of a cloudevent-trigger (or an equivalent consumer input) subscribing to a routed type."
  value = {
    for type in keys(var.dedicated_topics) : type => {
      for region in keys(var.regions) : region => google_pubsub_topic.dedicated["${region}-${type}"].name
    }
  }
}
