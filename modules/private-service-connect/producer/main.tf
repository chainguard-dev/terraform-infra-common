# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

terraform {
  required_version = ">= 1.9"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = ">= 8.7.0"
    }
  }
}

# Serverless NEG fronting the existing regional Cloud Run service.
resource "google_compute_region_network_endpoint_group" "this" {
  project               = var.project
  region                = var.region
  name                  = "${var.name}-neg"
  network_endpoint_type = "SERVERLESS"

  cloud_run {
    service = var.cloud_run_service_name
  }
}

# Regional internal-managed backend service pointing at the serverless NEG.
# Serverless NEG backends do not use health checks, and GCP rejects a
# configured timeout_sec for them ("Timeout sec is not supported for a backend
# service with Serverless network endpoint groups") — the effective request
# timeout on this path is the Cloud Run service's own request timeout, so tune
# that (regional-go-service request_timeout_seconds) for long-running backends.
resource "google_compute_region_backend_service" "this" {
  project               = var.project
  region                = var.region
  name                  = "${var.name}-backend"
  load_balancing_scheme = "INTERNAL_MANAGED"
  protocol              = "HTTP"

  backend {
    group = google_compute_region_network_endpoint_group.this.id
  }

  # Opt-in L7 access logging (var.enable_logging). Default off keeps existing
  # producers unchanged; when on, request logs flow to Cloud Logging and reach
  # the SIEM via the org all-load-balancer sink.
  dynamic "log_config" {
    for_each = var.enable_logging ? [1] : []
    content {
      enable      = true
      sample_rate = 1.0
    }
  }
}

# Regional URL map routing all traffic to the backend service.
resource "google_compute_region_url_map" "this" {
  project         = var.project
  region          = var.region
  name            = "${var.name}-urlmap"
  default_service = google_compute_region_backend_service.this.id
}

locals {
  # The frontend protocol: plain HTTP on :80 unless the caller hands us
  # Certificate Manager certificates to serve, then HTTPS on :443.
  https = length(var.tls_certificates) > 0
}

# Regional target HTTP proxy (the default frontend). Plain HTTP here is not a
# PSC limitation: TLS to the run.app backend is handled by the serverless NEG,
# and inbound authorization is enforced via Cloud Run invoker IAM (configured
# by the caller, not here). It is the historical shape, kept because an HTTPS
# frontend needs a certificate for a name consumers resolve to their own
# endpoint IP — see tls_certificates for that shape.
resource "google_compute_region_target_http_proxy" "this" {
  count = local.https ? 0 : 1

  project = var.project
  region  = var.region
  name    = "${var.name}-http-proxy"
  url_map = google_compute_region_url_map.this.id
}

# The frontend was unconditional before tls_certificates existed; keep the
# state address of every existing producer stable.
moved {
  from = google_compute_region_target_http_proxy.this
  to   = google_compute_region_target_http_proxy.this[0]
}

# Regional target HTTPS proxy: the frontend when the caller supplies
# Certificate Manager regional certificates (tls_certificates). Consumers
# dial the certificate's hostname, resolved to their PSC endpoint IP by a
# private DNS zone in their own VPC, so the handshake verifies against public
# roots with no trust distribution.
resource "google_compute_region_target_https_proxy" "this" {
  count = local.https ? 1 : 0

  project = var.project
  region  = var.region
  name    = "${var.name}-https-proxy"
  url_map = google_compute_region_url_map.this.id

  certificate_manager_certificates = var.tls_certificates
}

# Internal ALB frontend. The REGIONAL_MANAGED_PROXY subnet must exist in the
# region before this forwarding rule can be created; the caller passes its
# self-link so we can express that ordering explicitly.
#
# Changing allow_global_access forces this rule to be replaced (the provider
# treats the field as immutable), as does switching the frontend protocol
# (port_range 80 <-> 443). GCP also won't delete the old rule while a PSC
# service attachment references it, so the attachment is recreated alongside it
# — see the lifecycle block on google_compute_service_attachment.this below.
resource "google_compute_forwarding_rule" "this" {
  project               = var.project
  region                = var.region
  name                  = "${var.name}-fr"
  load_balancing_scheme = "INTERNAL_MANAGED"
  network               = var.network
  subnetwork            = var.subnetwork
  target                = local.https ? google_compute_region_target_https_proxy.this[0].id : google_compute_region_target_http_proxy.this[0].id
  port_range            = local.https ? "443" : "80"
  allow_global_access   = var.allow_global_access

  labels = var.labels

  depends_on = [var.proxy_only_subnet]
}

# PSC service attachment publishing the internal ALB to accepted consumers.
resource "google_compute_service_attachment" "this" {
  project               = var.project
  region                = var.region
  name                  = "${var.name}-sa"
  target_service        = google_compute_forwarding_rule.this.id
  connection_preference = "ACCEPT_MANUAL"
  nat_subnets           = var.psc_nat_subnets
  enable_proxy_protocol = false

  # Make accept-list removal an actual revocation. Updates in place.
  reconcile_connections = var.reconcile_connections

  dynamic "consumer_accept_lists" {
    for_each = var.consumer_accept_projects
    content {
      project_id_or_num = consumer_accept_lists.value
      connection_limit  = var.connection_limit
    }
  }

  dynamic "consumer_accept_lists" {
    for_each = var.consumer_accept_networks
    content {
      network_url      = consumer_accept_lists.value
      connection_limit = var.connection_limit
    }
  }

  # Recreate the attachment whenever the forwarding rule is replaced (an
  # allow_global_access change, or the HTTP -> HTTPS frontend switch). The
  # forwarding rule's self-link is name-stable, so target_service does not
  # change and the attachment would otherwise be left in place — but it
  # references (and so locks) the forwarding rule, blocking the rule's destroy
  # with "resourceInUseByAnotherResource". Replacing the attachment too makes
  # Terraform tear it down first (dependents are destroyed before their
  # dependencies), freeing the rule to be replaced, then recreates it.
  #
  # Watch the computed id: a replacement makes it unknown until apply, while
  # an in-place update retains it. Watching the whole resource also fires on
  # label updates, unnecessarily replacing the attachment and closing consumers.
  #
  # Deleting the attachment closes every connected consumer endpoint, and a
  # CLOSED PSC connection is terminal: the endpoint does not reconnect to the
  # recreated attachment (ACCEPT_MANUAL re-acceptance is for new connections),
  # and its forwarding rule shows no Terraform drift because the attachment's
  # self-link is unchanged. Every consumer must recreate its endpoint after
  # this applies (the consumer module's connection_generation input); the
  # seam is down for each consumer until it does. A live flip is therefore a
  # coordinated change with every accepted consumer, not a producer-only one.
  # https://cloud.google.com/vpc/docs/about-accessing-vpc-hosted-services-endpoints
  lifecycle {
    replace_triggered_by = [google_compute_forwarding_rule.this.id]
  }
}

# PSC traffic reaches the internal ALB from the attachment's NAT subnets, so
# those ranges are the only clients the VIP needs to admit.
data "google_compute_subnetwork" "psc_nat" {
  for_each  = var.lb_firewall == null ? toset([]) : toset(var.psc_nat_subnets)
  self_link = each.value
}

resource "google_compute_region_network_firewall_policy_rule" "lb_allow" {
  count = var.lb_firewall == null ? 0 : 1

  project         = coalesce(var.lb_firewall.project, var.project)
  region          = var.region
  firewall_policy = var.lb_firewall.policy
  rule_name       = "${var.name}-lb-allow"
  description     = "Admit PSC consumers (NAT subnets) to the ${var.name} internal ALB."
  priority        = var.lb_firewall.priority
  direction       = "INGRESS"
  action          = "allow"

  target_type             = "INTERNAL_MANAGED_LB"
  target_forwarding_rules = [google_compute_forwarding_rule.this.self_link]

  match {
    src_ip_ranges = concat([for s in data.google_compute_subnetwork.psc_nat : s.ip_cidr_range], var.lb_firewall.allowed_source_ranges)

    layer4_configs {
      ip_protocol = "tcp"
      ports       = [local.https ? "443" : "80"]
    }
  }
}

resource "google_compute_region_network_firewall_policy_rule" "lb_deny" {
  count = var.lb_firewall == null ? 0 : 1

  project         = coalesce(var.lb_firewall.project, var.project)
  region          = var.region
  firewall_policy = var.lb_firewall.policy
  rule_name       = "${var.name}-lb-deny"
  description     = "Close the ${var.name} internal ALB to every source but its PSC consumers."
  priority        = var.lb_firewall.priority + 1
  direction       = "INGRESS"
  action          = "deny"
  enable_logging  = true

  target_type             = "INTERNAL_MANAGED_LB"
  target_forwarding_rules = [google_compute_forwarding_rule.this.self_link]

  match {
    src_ip_ranges = ["0.0.0.0/0"]

    layer4_configs {
      ip_protocol = "all"
    }
  }
}
