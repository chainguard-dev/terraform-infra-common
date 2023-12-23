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
}

// Sign each of the application images.
resource "cosign_sign" "this" {
  for_each = var.containers
  image    = ko_build.this[each.key].image_ref
  conflict = "REPLACE"
}

// Build our otel-collector sidecar image.
module "otel-collector" {
  source = "../otel-collector"

  project_id      = var.project_id
  service_account = var.service_account
}

// Deploy the service into each of our regions.
resource "google_cloud_run_v2_service" "this" {
  for_each = var.regions

  provider = google-beta # For empty_dir
  project  = var.project_id
  name     = var.name
  location = each.key
  ingress  = var.ingress

  launch_stage = "BETA" // Needed for vpc_access below

  template {
    vpc_access {
      network_interfaces {
        network    = each.value.network
        subnetwork = each.value.subnet
      }
      egress = var.egress
      // TODO(mattmoor): When direct VPC egress supports network tags
      // for NAT egress, then we should incorporate those here.
    }

    service_account = var.service_account
    dynamic "containers" {
      for_each = var.containers
      content {
        image = cosign_sign.this[containers.key].signed_ref

        dynamic "ports" {
          for_each = containers.value.ports
          content {
            name           = ports.value.name
            container_port = ports.value.container_port
          }
        }

        dynamic "env" {
          for_each = containers.value.env
          content {
            name  = env.value.name
            value = env.value.value
          }
        }

        // Iterate over regional environment variables and look up the
        // appropriate value to pass to each region.
        dynamic "env" {
          for_each = containers.value.regional-env
          content {
            name  = env.value.name
            value = env.value.value[each.key]
          }
        }

        dynamic "volume_mounts" {
          for_each = containers.value.volume_mounts
          content {
            name       = volume_mounts.value.name
            mount_path = volume_mounts.value.mount_path
          }
        }
      }
    }
    containers { image = module.otel-collector.image }

    dynamic "volumes" {
      for_each = var.volumes
      content {
        name = volumes.value.name

        dynamic "secret" {
          for_each = volumes.value.secret != null ? { "" : volumes.value.secret } : {}
          content {
            secret = secret.value.secret
            dynamic "items" {
              for_each = secret.value.items
              content {
                version = items.value.version
                path    = items.value.path
              }
            }
          }
        }

        dynamic "empty_dir" {
          for_each = volumes.value.empty_dir != null ? { "" : volumes.value.empty_dir } : {}
          content {
            medium = empty_dir.value.medium
          }
        }
      }
    }
  }
}
