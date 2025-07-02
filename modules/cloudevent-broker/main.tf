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

  squad_label = var.squad != "" ? {
    squad = var.squad
    team  = var.squad
  } : {}
  product_label = var.product != "" ? {
    product = var.product
  } : {}

  merged_labels = merge(local.default_labels, local.squad_label, local.product_label, var.labels)
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
