/*
Copyright 2022 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Create a shared secret to have the uptime check pass to the
// Cloud Run app as an "Authorization" header to keep ~anyone
// from being able to use our prober endpoints to indirectly
// DoS our SaaS.
resource "random_password" "secret" {
  length           = 64
  special          = true
  override_special = "!#$%&*()-_=+[]{}<>:?"
}

locals {
  service_name = "prb-${substr(var.name, 0, 45)}" // use a common prefix so that they group together.
}

module "this" {
  source = "../regional-go-service"

  project_id = var.project_id
  name       = local.service_name
  regions    = var.regions
  scaling    = var.scaling

  squad         = var.squad
  require_squad = var.require_squad

  // If we're using GCLB then disallow external traffic,
  // otherwise allow the prober URI to be used directly.
  ingress = local.use_gclb ? "INGRESS_TRAFFIC_INTERNAL_LOAD_BALANCER" : "INGRESS_TRAFFIC_ALL"

  // Different probers have different egress requirements.
  egress = var.egress

  request_timeout_seconds = var.service_timeout_seconds

  deletion_protection = var.deletion_protection

  service_account = var.service_account
  containers = {
    "prober" = {
      source = {
        working_dir = var.working_dir
        importpath  = var.importpath
        base_image  = var.base_image
      }
      ports = [{ container_port = 8080 }]
      env = concat([
        {
          // This is a shared secret with the uptime check, which must be
          // passed in an Authorization header for the probe to do work.
          name  = "AUTHORIZATION"
          value = random_password.secret.result
        }
        ],
        [for k, v in var.env : { name = k, value = v }],
        [
          for k, v in var.secret_env : {
            name = k,
            value_source = {
              secret_key_ref = {
                secret  = v
                version = "latest"
              }
            }
          }
      ])
      regional-env = var.regional-env
      resources = {
        limits = {
          cpu    = var.cpu
          memory = var.memory
        }
        requests = {
          cpu    = var.cpu
          memory = var.memory
        }
      }
    }
  }

  enable_profiler = var.enable_profiler

  notification_channels = var.notification_channels
}

data "google_cloud_run_v2_service" "this" {
  count      = local.use_gclb ? 0 : 1
  depends_on = [module.this]

  project  = var.project_id
  location = keys(var.regions)[0]
  name     = local.service_name
}

// This is the uptime check, which will send traffic to the Cloud Run
// application every few minutes (from several locations) to ensure
// things are operating as expected.
resource "google_monitoring_uptime_check_config" "regional_uptime_check" {
  count = local.use_gclb ? 0 : 1

  display_name     = "${var.name}-uptime-regional"
  project          = var.project_id
  timeout          = var.timeout
  period           = var.period
  selected_regions = var.selected_regions

  http_check {
    path         = "/"
    port         = "443"
    use_ssl      = true
    validate_ssl = true

    // Pass the shared secret as an Authorization header.
    headers = {
      "Authorization" = random_password.secret.result
    }
  }

  monitored_resource {
    labels = {
      // Strip the scheme and path off of the Cloud Run URL.
      host       = split("/", data.google_cloud_run_v2_service.this[0].uri)[2]
      project_id = var.project_id
    }

    type = "uptime_url"
  }

  lifecycle {
    # We must create any replacement uptime checks before
    # we tear this check down.
    create_before_destroy = true
  }
}

// This is the uptime check, which will send traffic to the GCLB
// address every few minutes (from several locations) to ensure
// things are operating as expected.
resource "google_monitoring_uptime_check_config" "global_uptime_check" {
  count = local.use_gclb ? 1 : 0

  display_name     = "${var.name}-uptime-global"
  project          = var.project_id
  timeout          = var.timeout
  period           = var.period
  selected_regions = var.selected_regions

  http_check {
    path         = "/"
    port         = "443"
    use_ssl      = true
    validate_ssl = true

    // Pass the shared secret as an Authorization header.
    headers = {
      "Authorization" = random_password.secret.result
    }
  }

  monitored_resource {
    labels = {
      host       = "${var.name}-prober.${var.domain}."
      project_id = var.project_id
    }

    type = "uptime_url"
  }

  lifecycle {
    # We must create any replacement uptime checks before
    # we tear this check down.
    create_before_destroy = true
  }
}
