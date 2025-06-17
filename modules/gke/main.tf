terraform {
  required_providers {
    google = {
      source = "hashicorp/google"
    }
  }
}

# The default service account applied to all cluster node pools
resource "google_service_account" "cluster_default" {
  account_id   = "${var.name}-gke-default"
  display_name = "${var.name} GKE Default"
  project      = var.project
}

resource "google_project_iam_member" "cluster" {
  for_each = merge({
    # Read access to the project GCR
    "gcr" = "roles/storage.objectViewer"
    # Read access to the project ACR
    "acr" = "roles/artifactregistry.reader"
    # Log writer access
    "log-writer" = "roles/logging.logWriter"
    # Metrics writer access
    "metrics-writer" = "roles/monitoring.metricWriter"
    # Monitoring viewer access
    "monitoring-viewer" = "roles/monitoring.viewer"
  }, var.extra_roles)

  project = var.project
  role    = each.value
  member  = "serviceAccount:${google_service_account.cluster_default.email}"
}

locals {
  default_labels = {
    "gke" = var.name
  }

  squad_label = {
    "squad" = var.squad
    "team"  = var.squad
  }
}

resource "google_container_cluster" "this" {
  name    = var.name
  project = var.project

  network    = var.network
  subnetwork = var.subnetwork

  location       = var.region
  node_locations = var.zones

  deletion_protection = var.deletion_protection

  enable_intranode_visibility = true

  remove_default_node_pool = true
  initial_node_count       = 1

  resource_labels = var.labels

  # Use Dataplane V2 (eBPF based networking)
  datapath_provider = "ADVANCED_DATAPATH"

  networking_mode = "VPC_NATIVE"
  // Keeping this empty means GKE handles the secondary pod/service CIDR creation
  ip_allocation_policy {}

  workload_identity_config {
    workload_pool = "${var.project}.svc.id.goog"
  }

  release_channel {
    # NOTE: Toggle to "RAPID" when we want to start playing with things like gcsfuse
    channel = var.release_channel
  }

  # Configured with separate node_pool resources
  # node_config {}

  dynamic "cluster_autoscaling" {
    for_each = var.cluster_autoscaling == false ? [] : ["placeholder"]

    content {
      enabled = var.cluster_autoscaling
      resource_limits {
        resource_type = var.cluster_autoscaling_cpu_limits.resource_type
        minimum       = var.cluster_autoscaling_cpu_limits.minimum
        maximum       = var.cluster_autoscaling_cpu_limits.maximum
      }
      resource_limits {
        resource_type = var.cluster_autoscaling_memory_limits.resource_type
        minimum       = var.cluster_autoscaling_memory_limits.minimum
        maximum       = var.cluster_autoscaling_memory_limits.maximum
      }
      dynamic "auto_provisioning_defaults" {
        for_each = var.cluster_autoscaling_provisioning_defaults == null ? [] : ["placeholder"]

        content {
          service_account = google_service_account.cluster_default.email
          disk_size       = var.cluster_autoscaling_provisioning_defaults.disk_size
          disk_type       = var.cluster_autoscaling_provisioning_defaults.disk_type

          dynamic "shielded_instance_config" {
            for_each = var.cluster_autoscaling_provisioning_defaults.shielded_instance_config == null ? [] : ["placeholder"]

            content {
              enable_secure_boot          = var.cluster_autoscaling_provisioning_defaults.shielded_instance_config.enable_secure_boot
              enable_integrity_monitoring = var.cluster_autoscaling_provisioning_defaults.shielded_instance_config.enable_integrity_monitoring
            }
          }
          dynamic "management" {
            for_each = var.cluster_autoscaling_provisioning_defaults.management == null ? [] : ["placeholder"]

            content {
              auto_upgrade = var.cluster_autoscaling_provisioning_defaults.management.auto_upgrade
              auto_repair  = var.cluster_autoscaling_provisioning_defaults.management.auto_repair
            }
          }
        }
      }
      autoscaling_profile = var.cluster_autoscaling_profile
    }
  }

  master_authorized_networks_config {
    # gcp_public_cidrs_access_enabled = true
    cidr_blocks {
      display_name = "Everywhere"
      cidr_block   = "0.0.0.0/0"
    }

    # TODO: Pin this to https://api.github.com/meta
    # Github recommends against doing this, so maybe there's a more effective way, perhaps a certain scale with a tail?
    # cidr_blocks {}
  }

  private_cluster_config {
    enable_private_nodes = var.enable_private_nodes
    master_global_access_config {
      enabled = true
    }
    # This doesn't do what you think it does
    # private_endpoint_subnetwork = var.subnetwork
  }

  dns_config {
    # Enable more efficient DNS resolution by leveraging the GCP backplane (instead of kube-dns)
    # Technically this adds cloud DNS billing, but the cost is negligible
    # https://cloud.google.com/kubernetes-engine/docs/how-to/cloud-dns
    cluster_dns       = "CLOUD_DNS"
    cluster_dns_scope = "CLUSTER_SCOPE"
  }

  # TODO: These probably could be configurable
  addons_config {
    http_load_balancing {
      disabled = false
    }
    gke_backup_agent_config {
      enabled = false
    }
    config_connector_config {
      enabled = false
    }
    gcs_fuse_csi_driver_config {
      enabled = true
    }
  }

  monitoring_config {
    enable_components = ["SYSTEM_COMPONENTS", "APISERVER", "SCHEDULER", "CONTROLLER_MANAGER", "STORAGE", "POD"]
    managed_prometheus { enabled = true }

  }

  # This can't hurt... right?
  cost_management_config {
    enabled = true
  }

  dynamic "resource_usage_export_config" {
    for_each = var.resource_usage_export_config.bigquery_dataset_id == "" ? [] : ["placeholder"]

    content {
      enable_network_egress_metering       = var.resource_usage_export_config.enable_network_egress_metering
      enable_resource_consumption_metering = var.resource_usage_export_config.enable_resource_consumption_metering
      bigquery_destination {
        dataset_id = var.resource_usage_export_config.bigquery_dataset_id
      }
    }
  }

  timeouts {
    create = "30m"
    update = "30m"
    delete = "30m"
  }

  lifecycle {
    # https://github.com/hashicorp/terraform-provider-google/issues/6901
    ignore_changes = [initial_node_count]
  }

  depends_on = [google_service_account.cluster_default]
}

resource "google_container_node_pool" "pools" {
  for_each = var.pools
  provider = google-beta

  name     = each.key
  cluster  = google_container_cluster.this.name
  project  = var.project
  location = google_container_cluster.this.location

  dynamic "network_config" {
    for_each = each.value.network_config != null ? [1] : []
    content {
      enable_private_nodes = each.value.network_config.enable_private_nodes
      create_pod_range     = each.value.network_config.create_pod_range
      pod_ipv4_cidr_block  = each.value.network_config.pod_ipv4_cidr_block
    }
  }

  node_config {
    service_account = google_service_account.cluster_default.email
    image_type      = "COS_CONTAINERD"
    machine_type    = each.value.machine_type
    workload_metadata_config {
      # Run the GKE metadata server on these nodes (required for workload identity)
      mode = "GKE_METADATA"
    }
    metadata = {
      disable-legacy-endpoints = true
      block-project-ssh-keys   = true
    }

    disk_type    = each.value.disk_type
    disk_size_gb = each.value.disk_size

    dynamic "ephemeral_storage_local_ssd_config" {
      for_each = each.value.ephemeral_storage_local_ssd_count > 0 ? [1] : []
      content {
        local_ssd_count = each.value.ephemeral_storage_local_ssd_count
      }
    }

    # Don't set legacy scopes
    # oauth_scopes = []

    # Enable google vNIC driver
    gvnic {
      enabled = true
    }

    # Enable google container filesystem (required for image streaming)
    gcfs_config {
      enabled = true
    }

    dynamic "sandbox_config" {
      for_each = each.value.gvisor ? [1] : []
      content {
        sandbox_type = "gvisor"
      }
    }

    spot            = each.value.spot
    labels          = each.value.labels
    resource_labels = merge(local.default_labels, local.squad_label, var.labels)

    dynamic "taint" {
      for_each = each.value.taints
      content {
        key    = taint.value.key
        value  = taint.value.value
        effect = taint.value.effect
      }
    }
  }

  autoscaling {
    min_node_count = each.value.min_node_count
    max_node_count = each.value.max_node_count
  }

  management {
    auto_repair  = true
    auto_upgrade = true
  }
}

# Allow GKE master to hit non 443 ports for webhook/admission controllers
#
# https://github.com/kubernetes/kubernetes/issues/79739
resource "google_compute_firewall" "master_webhook" {
  project = var.project
  network = var.network

  name        = "${var.name}-master-webhook"
  description = "Allow GKE master to hit non 443 ports for webhook/admission controllers"
  direction   = "INGRESS"

  source_ranges = ["${google_container_cluster.this.endpoint}/32"]
  source_tags   = []
  target_tags   = ["gke-${google_container_cluster.this.name}"]

  allow {
    protocol = "tcp"
    ports = [
      "8443",
      "9443",
      "15017",
    ]
  }

  depends_on = [google_container_cluster.this]
}
