/*
Copyright 2022 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Create a shared secret to have the uptime check pass to the
// Cloud Run app as an "Authorization" header to keep ~anyone
// from being able to use our prober endpoints to indirectly
// DoS our SaaS. With service_agent_auth, Cloud Run IAM plays
// that role instead and no secret exists at all.
resource "random_password" "secret" {
  count = var.service_agent_auth ? 0 : 1

  length           = 64
  special          = true
  override_special = "!#$%&*()-_=+[]{}<>:?"
}

locals {
  service_name = "prb-${substr(var.name, 0, 45)}" // use a common prefix so that they group together.

  // service_agent_auth relies on Cloud Run IAM in front of the service. A
  // GCLB terminates the uptime check's identity rather than forwarding it
  // to the backing service, so multi-region probers must keep the
  // shared-secret gate.
  service_agent_auth_guard = (var.service_agent_auth && local.use_gclb) ? tobool("service_agent_auth is not supported for multi-region (GCLB) probers") : true
}

module "this" {
  source             = "../regional-go-service"
  observability_role = var.observability_role

  project_id = var.project_id
  name       = local.service_name
  regions    = var.regions
  scaling    = var.scaling

  team    = var.team
  product = var.product

  // If we're using GCLB then disallow external traffic,
  // otherwise allow the prober URI to be used directly.
  ingress = local.use_gclb ? "INGRESS_TRAFFIC_INTERNAL_LOAD_BALANCER" : "INGRESS_TRAFFIC_ALL"

  // Different probers have different egress requirements.
  egress = var.egress

  // With service_agent_auth, the service accepts only IAM-authenticated
  // invocations; the uptime check's service agent is granted run.invoker
  // below.
  require_authenticated_invocations = var.service_agent_auth

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
      env = concat(
        // This is a shared secret with the uptime check, which must be
        // passed in an Authorization header for the probe to do work.
        // Under service_agent_auth, Cloud Run IAM gates the endpoint and
        // no secret is injected.
        var.service_agent_auth ? [] : [{
          name  = "AUTHORIZATION"
          value = random_password.secret[0].result
        }],
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
      regional-env      = var.regional-env
      regional-cpu-idle = var.cpu_idle
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

  launch_stage = var.launch_stage

  notification_channels = var.notification_channels

  resource_manager_tags = var.resource_manager_tags
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

    // Pass the shared secret as an Authorization header, unless Cloud Run
    // IAM gates the endpoint instead.
    headers = var.service_agent_auth ? {} : {
      "Authorization" = random_password.secret[0].result
    }

    // Authenticate as the Cloud Monitoring service agent, which holds
    // run.invoker on the service.
    dynamic "service_agent_authentication" {
      for_each = var.service_agent_auth ? [1] : []
      content {
        type = "OIDC_TOKEN"
      }
    }
  }

  // Service-agent authentication is only supported against a Cloud Run
  // resource, not an arbitrary URL, so that mode monitors the service
  // directly instead of its uptime_url.
  monitored_resource {
    type = var.service_agent_auth ? "cloud_run_revision" : "uptime_url"

    labels = var.service_agent_auth ? {
      project_id   = var.project_id
      service_name = local.service_name
      location     = keys(var.regions)[0]
      } : {
      // Strip the scheme and path off of the Cloud Run URL.
      host       = split("/", data.google_cloud_run_v2_service.this[0].uri)[2]
      project_id = var.project_id
    }
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

    // Pass the shared secret as an Authorization header. (This check only
    // exists for multi-region probers, where service_agent_auth is
    // unsupported — see service_agent_auth_guard.)
    headers = {
      "Authorization" = random_password.secret[0].result
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

// With service_agent_auth, the uptime check authenticates as Monitoring's
// service agent, which must be allowed to invoke the otherwise IAM-gated
// service in each region. The agent that signs the check's OIDC token is
// the one this resource returns — service-N@gcp-sa-monitoring-notification,
// NOT service-N@gcp-sa-monitoring (observed on the wire:
// https://seankhliao.com/blog/12024-07-20-gcp-uptime-service-agent-auth/).
// The agent is also created lazily by GCP, so provision it explicitly
// rather than assuming a prior uptime check or notification channel
// already forced it into existence.
resource "google_project_service_identity" "monitoring" {
  count    = var.service_agent_auth ? 1 : 0
  provider = google-beta

  project = var.project_id
  service = "monitoring.googleapis.com"
}

resource "google_cloud_run_v2_service_iam_member" "uptime-check-invoker" {
  for_each = var.service_agent_auth ? var.regions : {}

  project  = var.project_id
  location = each.key
  name     = local.service_name
  role     = "roles/run.invoker"
  member   = "serviceAccount:${google_project_service_identity.monitoring[0].email}"

  depends_on = [module.this]
}
