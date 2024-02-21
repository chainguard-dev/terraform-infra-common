terraform {
  required_providers {
    ko     = { source = "ko-build/ko" }
    cosign = { source = "chainguard-dev/cosign" }
  }
}

module "audit-serviceaccount" {
  source = "../audit-serviceaccount"

  project_id      = var.project_id
  service-account = var.service_account

  # The absence of authorized identities here means that
  # nothing is authorized to act as this service account.
  # Note: Cloud Run's usage doesn't show up in the audit logs.

  notification_channels = var.notification_channels
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
    scaling {
      min_instance_count = var.scaling.min_instances
      max_instance_count = var.scaling.max_instances
    }
    max_instance_request_concurrency = var.scaling.max_instance_request_concurrency
    execution_environment            = var.execution_environment
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
    timeout         = "${var.request_timeout_seconds}s"
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
            limits   = resources.value.limits
            cpu_idle = resources.value.cpu_idle
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

    // Regional volumes
    dynamic "volumes" {
      for_each = var.regional-volumes
      content {
        name = volumes.value.name

        dynamic "gcs" {
          for_each = length(volumes.value.gcs) > 0 ? { "" : volumes.value.gcs[each.key] } : {}
          content {
            bucket    = gcs.value.bucket
            read_only = gcs.value.read_only
          }
        }
        dynamic "nfs" {
          for_each = length(volumes.value.nfs) > 0 ? { "" : volumes.value.nfs[each.key] } : {}
          content {
            server    = nfs.value.server
            path      = nfs.value.path
            read_only = nfs.value.read_only
          }
        }
      }
    }
  }
}

// Get a project number for this project ID.
data "google_project" "project" { project_id = var.project_id }

// What identity is deploying this?
data "google_client_openid_userinfo" "me" {}

// Create an alert policy to notify if the service is accessed by an unauthorized entity.
resource "google_monitoring_alert_policy" "anomalous-service-access" {
  # In the absence of data, incident will auto-close after an hour
  alert_strategy {
    auto_close = "3600s"

    notification_rate_limit {
      period = "3600s" // re-alert hourly if condition still valid.
    }
  }

  display_name = "Abnormal Service Access: ${var.name}"
  combiner     = "OR"

  conditions {
    display_name = "Abnormal Service Access: ${var.name}"

    condition_matched_log {
      filter = <<EOT
      logName="projects/${var.project_id}/logs/cloudaudit.googleapis.com%2Fdata_access"
      protoPayload.serviceName="run.googleapis.com"
      protoPayload.resourceName=("${join("\" OR \"", concat([
        "namespaces/${var.project_id}/services/${var.name}"
      ],
      [
        for region in keys(var.regions) : "projects/${var.project_id}/locations/${region}/services/${var.name}"
      ]))}")

      -- Allow CI to reconcile services and their IAM policies.
      -(
        protoPayload.authenticationInfo.principalEmail="${data.google_client_openid_userinfo.me.email}"
        protoPayload.methodName=("${join("\" OR \"", [
          "google.cloud.run.v2.Services.GetService",
          "google.cloud.run.v2.Services.GetIamPolicy",
          "google.cloud.run.v2.Services.UpdateService",
          "google.cloud.run.v2.Services.SetIamPolicy",
        ])}")
      )
      EOT
    }
  }

  # TODO(mattmoor): Enable notifications once this stabilizes.
  # notification_channels = var.notification_channels

  enabled = "true"
  project = var.project_id
}

// When the service is behind a load balancer, then it is publicly exposed and responsible
// for handling its own authentication.
resource "google_cloud_run_v2_service_iam_member" "public-services-are-unauthenticated" {
  for_each = var.ingress != "INGRESS_TRAFFIC_INTERNAL_ONLY" ? var.regions : {}

  // Ensure that the service exists before attempting to expose things publicly.
  depends_on = [google_cloud_run_v2_service.this]

  project  = var.project_id
  location = each.key
  name     = google_cloud_run_v2_service.this[each.key].name
  role     = "roles/run.invoker"
  member   = "allUsers"
}
