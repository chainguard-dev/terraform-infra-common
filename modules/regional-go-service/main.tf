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

  // We won't be able to deploy if the service account doesn't have access to the network.
  depends_on = [google_compute_subnetwork_iam_member.member]

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
        args  = containers.value.args

        dynamic "ports" {
          for_each = containers.value.ports
          content {
            name           = ports.value.name
            container_port = ports.value.container_port
          }
        }

        dynamic "resources" {
          for_each = containers.value.resources != null ? { "" : containers.value.resources } : {}
          content {
            limits = resources.value.limits
          }
        }

        dynamic "env" {
          for_each = containers.value.env
          content {
            name  = env.value.name
            value = env.value.value
            dynamic "value_source" {
              for_each = env.value.value_source != null ? { "" : env.value.value_source } : {}
              content {
                secret_key_ref {
                  secret  = value_source.value.secret_key_ref.secret
                  version = value_source.value.secret_key_ref.version
                }
              }
            }
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

// When the service is behind a load balancer, then it is publicly exposed and responsible
// for handling its own authentication.
resource "google_cloud_run_v2_service_iam_member" "public-services-are-unauthenticated" {
  for_each = var.ingress == "INGRESS_TRAFFIC_INTERNAL_LOAD_BALANCER" ? var.regions : {}

  // Ensure that the service exists before attempting to expose things publicly.
  depends_on = [google_cloud_run_v2_service.this]

  project  = var.project_id
  location = each.key
  name     = google_cloud_run_v2_service.this[each.key].name
  role     = "roles/run.invoker"
  member   = "allUsers"
}

// Grant service account access to use subnet. This is typically granted with roles/run.serviceAgent,
// but that role does not necessarily grant access if the network resides in another project.
// See https://cloud.google.com/run/docs/configuring/vpc-direct-vpc#direct-vpc-service for more details.
resource "google_compute_subnetwork_iam_member" "member" {
  for_each = var.regions

  // If not set, provider project should be used.
  project    = var.network_project
  region     = each.key
  subnetwork = each.value.subnet
  role       = "roles/compute.networkUser"
  member     = "serviceAccount:${var.service_account}"
}
