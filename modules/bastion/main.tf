/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

resource "google_project_service" "os_config_api" {
  project            = var.project_id
  service            = "osconfig.googleapis.com"
  disable_on_destroy = false
}

locals {
  // Derive region from zone (e.g. us-central1-a -> us-central1)
  region        = join("-", slice(split("-", var.zone), 0, 2))
  instance_name = substr(var.name, 0, 63) # GCE name limit
  instance_tag  = var.name

  // Labels
  default_labels = {
    basename(abspath(path.module)) = var.name
    terraform-module               = basename(abspath(path.module))
  }

  squad_label = {
    squad = var.squad
    team  = var.squad
  }

  product_label = var.product != "" ? {
    product = var.product
  } : {}

  merged_labels = merge(local.default_labels, local.squad_label, local.product_label)

  // Split patch time HH:MM into components
  patch_hour   = tonumber(split(":", var.patch_time_utc)[0])
  patch_minute = tonumber(split(":", var.patch_time_utc)[1])

  // Optional Cloud SQL Auth Proxy binary install
  proxy_install_script = <<-PROXY
      # ------------------------------------------------------
      # Install Cloud SQL Auth Proxy v2 (binary only)
      # ------------------------------------------------------
      curl -sSL -o /usr/local/bin/cloud-sql-proxy \
        https://storage.googleapis.com/cloud-sql-connectors/cloud-sql-proxy/v2.16.0/cloud-sql-proxy.linux.amd64
      chmod +x /usr/local/bin/cloud-sql-proxy
  PROXY

  bastion_startup = join("\n", compact([
    <<-SCRIPT
      #!/usr/bin/env bash
      set -euo pipefail
    SCRIPT
    ,

    <<-AUDIT
      apt-get update -y
      apt-get install -y auditd
      systemctl enable --now auditd.service

      cat > /etc/audit/rules.d/99-bastion.rules <<'AUDITD'
      -w /usr/bin/ -p x -k execs
      -w /bin/ -p x -k execs
      -a always,exit -F arch=b64 -S execve -k execs
      AUDITD

      augenrules --load
    AUDIT
    ,

    var.startup_script,

    <<-COMMON
      # ------------------------------------------------------
      # Install Google Cloud Ops Agent for logging & metrics
      # ------------------------------------------------------
      curl -sSO https://dl.google.com/cloudagents/add-google-cloud-ops-agent-repo.sh
      bash add-google-cloud-ops-agent-repo.sh --also-install
    COMMON
    ,

    // Optionally install Cloud SQL Auth Proxy
    var.install_sql_proxy ? local.proxy_install_script : "",
  ]))
}

// -----------------------------------------------------------------------------
// Compute Engine VM
// -----------------------------------------------------------------------------

resource "google_compute_instance" "bastion" {
  name         = local.instance_name
  project      = var.project_id
  zone         = var.zone
  machine_type = var.machine_type

  boot_disk {
    initialize_params {
      image = "projects/debian-cloud/global/images/family/debian-12"
    }
  }

  shielded_instance_config {
    enable_secure_boot          = true
    enable_vtpm                 = true
    enable_integrity_monitoring = true
  }

  network_interface {
    nic_type           = "GVNIC"
    network            = var.network
    subnetwork         = var.subnetwork
    subnetwork_project = var.project_id
  }

  service_account {
    email = google_service_account.bastion_sa.email
    scopes = concat(
      var.install_sql_proxy ? ["https://www.googleapis.com/auth/sqlservice.admin"] : [],
      [
        "https://www.googleapis.com/auth/logging.write",
        "https://www.googleapis.com/auth/monitoring.write",
      ]
    )
  }

  metadata = {
    enable-oslogin         = "TRUE"
    block-project-ssh-keys = "TRUE"
    enable-oslogin-2fa     = "TRUE"
  }

  metadata_startup_script = local.bastion_startup

  tags = [local.instance_tag]

  allow_stopping_for_update = true

  scheduling {
    automatic_restart = true
  }

  deletion_protection = var.deletion_protection

  labels = local.merged_labels
}

// -----------------------------------------------------------------------------
// IAP-only SSH firewall rule
// -----------------------------------------------------------------------------

resource "google_compute_firewall" "iap_ssh" {
  name    = "allow-iap-ssh-${var.name}"
  project = var.project_id
  network = var.network

  direction = "INGRESS"
  priority  = 1000

  source_ranges = ["35.235.240.0/20"]
  target_tags   = [local.instance_tag]

  allow {
    protocol = "tcp"
    ports    = ["22"]
  }
}

// -----------------------------------------------------------------------------
// OS Config patch deployment
// -----------------------------------------------------------------------------

resource "google_os_config_patch_deployment" "bastion_patching" {
  project             = var.project_id
  patch_deployment_id = "${var.name}-weekly-patching"

  instance_filter {
    instance_name_prefixes = [google_compute_instance.bastion.name]
  }

  recurring_schedule {
    time_zone {
      id = "Etc/UTC"
    }

    weekly {
      day_of_week = upper(var.patch_day)
    }

    time_of_day {
      hours   = local.patch_hour
      minutes = local.patch_minute
      seconds = 0
      nanos   = 0
    }
  }

  patch_config {
    reboot_config = "ALWAYS"
  }

  depends_on = [google_project_service.os_config_api]
}

// -----------------------------------------------------------------------------
// Cloud NAT (optional)
// -----------------------------------------------------------------------------

resource "google_compute_router" "nat_router" {
  count   = var.enable_nat ? 1 : 0
  name    = "${var.project_id}-${var.name}-nat-router"
  project = var.project_id
  region  = local.region
  network = var.network
}

resource "google_compute_router_nat" "bastion_nat" {
  count   = var.enable_nat ? 1 : 0
  name    = "${var.project_id}-${var.name}-nat"
  project = var.project_id
  region  = local.region
  router  = google_compute_router.nat_router[count.index].name

  nat_ip_allocate_option             = "AUTO_ONLY"
  source_subnetwork_ip_ranges_to_nat = "LIST_OF_SUBNETWORKS"

  subnetwork {
    name                    = var.subnetwork
    source_ip_ranges_to_nat = ["ALL_IP_RANGES"]
  }

  log_config {
    enable = true
    filter = "ERRORS_ONLY"
  }
}

// -----------------------------------------------------------------------------
// Service account and IAM
// -----------------------------------------------------------------------------

resource "google_service_account" "bastion_sa" {
  project      = var.project_id
  account_id   = "${var.name}-sa"
  display_name = "Service account for ${var.name} bastion"
}

resource "google_project_iam_member" "bastion_sa_log_writer" {
  project = var.project_id
  role    = "roles/logging.logWriter"
  member  = "serviceAccount:${google_service_account.bastion_sa.email}"
}

resource "google_project_iam_member" "bastion_sa_metric_writer" {
  project = var.project_id
  role    = "roles/monitoring.metricWriter"
  member  = "serviceAccount:${google_service_account.bastion_sa.email}"
}

resource "google_project_iam_member" "bastion_extra_roles" {
  for_each = toset(var.extra_sa_roles)
  project  = var.project_id
  role     = each.value
  member   = "serviceAccount:${google_service_account.bastion_sa.email}"
}

// -----------------------------------------------------------------------------
// Developer IAM access
// -----------------------------------------------------------------------------

resource "google_iap_tunnel_instance_iam_member" "dev_os_login" {
  for_each = toset(var.dev_principals)

  project  = var.project_id
  instance = google_compute_instance.bastion.name
  role     = "roles/compute.osLogin"
  member   = each.key
}

resource "google_iap_tunnel_instance_iam_member" "dev_iap_tunnel_resource_accessor" {
  for_each = toset(var.dev_principals)

  project  = var.project_id
  instance = google_compute_instance.bastion.name
  role     = "roles/iap.tunnelResourceAccessor"
  member   = each.key
}

resource "google_iap_tunnel_instance_iam_member" "dev_service_account_user" {
  for_each = toset(var.dev_principals)

  project  = var.project_id
  instance = google_compute_instance.bastion.name
  role     = "roles/iam.serviceAccountUser"
  member   = each.key
}

resource "google_project_iam_member" "dev_cloudsql_client" {
  count   = var.install_sql_proxy ? length(distinct(var.dev_principals)) : 0
  project = var.project_id
  role    = "roles/cloudsql.client"
  member  = distinct(var.dev_principals)[count.index]
}

resource "google_project_iam_member" "dev_cloudsql_instance_user" {
  count   = var.install_sql_proxy ? length(distinct(var.dev_principals)) : 0
  project = var.project_id
  role    = "roles/cloudsql.instanceUser"
  member  = distinct(var.dev_principals)[count.index]
}
