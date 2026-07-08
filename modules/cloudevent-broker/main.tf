terraform {
  required_providers {
    ko     = { source = "ko-build/ko" }
    cosign = { source = "chainguard-dev/cosign" }
  }
}

locals {
  default_labels = {
    basename(abspath(path.module)) = var.name
    terraform-module               = basename(abspath(path.module))
  }

  squad_label = {
    squad = var.team
    team  = var.team
  }
  product_label = var.product != "" ? {
    product = var.product
  } : {}

  merged_labels = merge(local.default_labels, local.squad_label, local.product_label, var.labels)

  // Flatten dedicated_topics into per-region-per-type entries for for_each,
  // keyed "<region>-<type>" to match the ingress env and output lookups.
  dedicated-regional-types = merge([
    for region in keys(var.regions) : {
      for type in keys(var.dedicated_topics) :
      "${region}-${type}" => { region = region, type = type }
    }
  ]...)

  // Types whose route is enabled; the ingress routing table (built in
  // ingress.tf) is derived from these. Empty when nothing is routed yet.
  routed-types = { for type, cfg in var.dedicated_topics : type => cfg if cfg.route }
}

resource "google_pubsub_topic" "this" {
  for_each = var.regions

  name   = "${var.name}-${each.key}"
  labels = local.merged_labels

  // TODO: Tune this and/or make it configurable?
  message_retention_duration = "600s"

  message_storage_policy {
    allowed_persistence_regions = [each.key]
  }
}

// Dedicated per-region topics for high-volume, low-consumer event types routed
// off the shared firehose. Created for every declared type (including
// route=false) so consumers can subscribe before the ingress routes to them.
resource "google_pubsub_topic" "dedicated" {
  for_each = local.dedicated-regional-types

  name   = "${var.name}-${replace(each.value.type, ".", "-")}-${each.value.region}"
  labels = local.merged_labels

  message_retention_duration = var.dedicated_topics[each.value.type].message_retention_duration

  message_storage_policy {
    allowed_persistence_regions = [each.value.region]
  }
}
