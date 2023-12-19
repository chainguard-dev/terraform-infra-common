terraform {
  required_providers {
    ko     = { source = "ko-build/ko" }
    cosign = { source = "chainguard-dev/cosign" }
  }
}

resource "google_pubsub_topic" "this" {
  for_each = var.regions

  name = "${var.name}-${each.key}"

  // TODO: Tune this and/or make it configurable?
  message_retention_duration = "600s"
}
