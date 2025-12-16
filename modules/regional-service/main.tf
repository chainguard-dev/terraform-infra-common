resource "google_project_iam_member" "metrics-writer" {
  project = var.project_id
  role    = "roles/monitoring.metricWriter"
  member  = "serviceAccount:${var.service_account}"
}

resource "google_project_iam_member" "trace-writer" {
  project = var.project_id
  role    = "roles/cloudtrace.agent"
  member  = "serviceAccount:${var.service_account}"
}

resource "google_project_iam_member" "profiler-writer" {
  project = var.project_id
  role    = "roles/cloudprofiler.agent"
  member  = "serviceAccount:${var.service_account}"
}

locals {
  // Pull out the containers from var.containers that has no port
  sidecars = {
    for key, value in var.containers : key => value if length(value.ports) == 0
  }
  // Pull out the main container from var.containers that has a port.
  // There should be only one of them, but using a map to make it easier to
  // iterate over and look up the ko_builds.
  has_port = {
    for key, value in var.containers : key => value if length(value.ports) > 0
  }

  default_labels = {
    basename(abspath(path.module)) = var.name
    terraform-module               = basename(abspath(path.module))
  }

  squad_label = {
    "squad" : var.team
    "team" : var.team
  }

  product_label = var.product != "" ? {
    product = var.product
  } : {}

  main_container_idx = keys(local.has_port)[0]
  main_container     = local.has_port[local.main_container_idx]
}

check "exactly_one_main_container" {
  assert {
    condition     = length(local.has_port) == 1
    error_message = "Exactly one container with ports must be specified."
  }
}

// Deploy the service into each of our regions.
resource "google_cloud_run_v2_service" "this" {
  for_each = var.regions

  provider = google-beta # For empty_dir
  project  = var.project_id
  name     = var.name
  location = each.key
  labels   = merge(var.labels, local.default_labels, local.squad_label, local.product_label)
  ingress  = var.ingress

  deletion_protection = var.deletion_protection

  template {
    scaling {
      min_instance_count = var.scaling.min_instances
      max_instance_count = var.scaling.max_instances
    }
    max_instance_request_concurrency = var.scaling.max_instance_request_concurrency
    execution_environment            = var.execution_environment
    labels                           = merge(var.labels, local.default_labels, local.squad_label, local.product_label)

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

    // A main container has ports. It needs to go first to avoid a bug in the
    // Cloud Run terraform provider where omitting the port{} block does not
    // remove the port from the service.
    containers {
      image = local.main_container.image
      args  = local.main_container.args

      dynamic "ports" {
        for_each = local.main_container.ports
        content {
          name           = ports.value.name
          container_port = ports.value.container_port
        }
      }

      dynamic "resources" {
        for_each = local.main_container.resources != null ? { "" : local.main_container.resources } : {}
        content {
          limits            = resources.value.limits
          cpu_idle          = coalesce(resources.value.cpu_idle, lookup(local.main_container.regional-cpu-idle, each.key, true))
          startup_cpu_boost = resources.value.startup_cpu_boost
        }
      }

      dynamic "env" {
        for_each = local.main_container.env
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
        for_each = local.main_container.regional-env
        content {
          name  = env.value.name
          value = env.value.value[each.key]
        }
      }

      env {
        name  = "ENABLE_PROFILER"
        value = var.enable_profiler
      }

      dynamic "volume_mounts" {
        for_each = local.main_container.volume_mounts
        content {
          name       = volume_mounts.value.name
          mount_path = volume_mounts.value.mount_path
        }
      }
      dynamic "startup_probe" {
        for_each = local.main_container.startup_probe != null ? { "" : local.main_container.startup_probe } : {}
        content {
          dynamic "http_get" {
            for_each = startup_probe.value.http_get != null ? { "" : startup_probe.value.http_get } : {}
            content {
              path = http_get.value.path
              port = http_get.value.port
            }
          }
          dynamic "tcp_socket" {
            for_each = startup_probe.value.tcp_socket != null ? { "" : startup_probe.value.tcp_socket } : {}
            content {
              port = tcp_socket.value.port
            }
          }
          dynamic "grpc" {
            for_each = startup_probe.value.grpc != null ? { "" : startup_probe.value.grpc } : {}
            content {
              service = grpc.value.service
              port    = grpc.value.port
            }
          }

          initial_delay_seconds = startup_probe.value.initial_delay_seconds
          period_seconds        = startup_probe.value.period_seconds
          timeout_seconds       = startup_probe.value.timeout_seconds
          failure_threshold     = startup_probe.value.failure_threshold
        }
      }
      dynamic "liveness_probe" {
        for_each = local.main_container.liveness_probe != null ? { "" : local.main_container.liveness_probe } : {}
        content {
          dynamic "http_get" {
            for_each = liveness_probe.value.http_get != null ? { "" : liveness_probe.value.http_get } : {}
            content {
              path = http_get.value.path
              port = http_get.value.port
            }
          }
          dynamic "tcp_socket" {
            for_each = liveness_probe.value.tcp_socket != null ? { "" : liveness_probe.value.tcp_socket } : {}
            content {
              port = tcp_socket.value.port
            }
          }
          dynamic "grpc" {
            for_each = liveness_probe.value.grpc != null ? { "" : liveness_probe.value.grpc } : {}
            content {
              service = grpc.value.service
              port    = grpc.value.port
            }
          }

          initial_delay_seconds = liveness_probe.value.initial_delay_seconds
          period_seconds        = liveness_probe.value.period_seconds
          timeout_seconds       = liveness_probe.value.timeout_seconds
          failure_threshold     = liveness_probe.value.failure_threshold
        }
      }
    }

    // Now the sidecar containers can be added.
    dynamic "containers" {
      for_each = local.sidecars
      content {
        image = containers.value.image
        args  = containers.value.args

        dynamic "resources" {
          for_each = containers.value.resources != null ? { "" : containers.value.resources } : {}
          content {
            limits            = resources.value.limits
            cpu_idle          = coalesce(resources.value.cpu_idle, lookup(containers.value.regional-cpu-idle, each.key, true))
            startup_cpu_boost = resources.value.startup_cpu_boost
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
    containers {
      image = var.otel_collector_image
      // config via env is an option; https://pkg.go.dev/go.opentelemetry.io/collector/service#section-readme
      args = ["--config=env:OTEL_CONFIG"]
      env {
        name = "OTEL_CONFIG"
        value = replace(replace(replace(file("${path.module}/otel-config/config.yaml"),
          "REPLACE_ME_TEAM", var.team),
          "REPLACE_ME_PROJECT_ID", var.project_id),
        "REPLACE_ME_SERVICE", var.name)
      }

      dynamic "resources" {
        for_each = var.otel_resources != null ? { "" : var.otel_resources } : {}
        content {
          limits            = resources.value.limits
          cpu_idle          = resources.value.cpu_idle
          startup_cpu_boost = resources.value.startup_cpu_boost
        }
      }
    }

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
            medium     = empty_dir.value.medium
            size_limit = empty_dir.value.size_limit
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
            bucket        = gcs.value.bucket
            read_only     = gcs.value.read_only
            mount_options = gcs.value.mount_options
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

  lifecycle {
    ignore_changes = [
      launch_stage,
      # GCP manages container names automatically
      # Supporting up to 5 total containers (main + sidecars + otel)
      template[0].containers[0].name,
      template[0].containers[1].name,
      template[0].containers[2].name,
      template[0].containers[3].name,
      template[0].containers[4].name,
    ]
  }
}

// Get a project number for this project ID.
data "google_project" "project" { project_id = var.project_id }

// What identity is deploying this?
data "google_client_openid_userinfo" "me" {}

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

module "slo" {
  count = var.slo.enable ? 1 : 0

  source = "../slo"

  project_id   = var.project_id
  service_name = var.name
  service_type = "CLOUD_RUN"

  regions = keys(var.regions)

  slo = var.slo

  notification_channels = var.notification_channels
}
