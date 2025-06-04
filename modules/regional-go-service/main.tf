terraform {
  required_providers {
    ko     = { source = "ko-build/ko" }
    cosign = { source = "chainguard-dev/cosign" }
  }
}

// Build each of the application images from source.
resource "ko_build" "this" {
  for_each    = var.containers
  base_image  = each.value.source.base_image
  working_dir = each.value.source.working_dir
  importpath  = each.value.source.importpath
  env         = each.value.source.env
}

resource "cosign_sign" "this" {
  for_each = var.containers
  image    = ko_build.this[each.key].image_ref
  conflict = "REPLACE"
}

module "this" {
  source = "../regional-service"

  project_id = var.project_id
  name       = var.name
  regions    = var.regions
  ingress    = var.ingress
  egress     = var.egress

  deletion_protection = var.deletion_protection

  service_account = var.service_account
  containers = {
    for name, container in var.containers : name => {
      image          = cosign_sign.this[name].signed_ref
      args           = container.args
      ports          = container.ports
      resources      = container.resources
      env            = container.env
      regional-env   = container.regional-env
      volume_mounts  = container.volume_mounts
      startup_probe  = container.startup_probe
      liveness_probe = container.liveness_probe
    }
  }

  labels           = var.labels
  squad            = var.squad
  require_squad    = var.require_squad
  scaling          = var.scaling
  volumes          = var.volumes
  regional-volumes = var.regional-volumes
  enable_profiler  = var.enable_profiler
  otel_resources   = var.otel_resources

  request_timeout_seconds = var.request_timeout_seconds
  execution_environment   = var.execution_environment

  notification_channels = var.notification_channels
}
