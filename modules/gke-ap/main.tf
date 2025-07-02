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
  display_name = "${var.name} GKE AP Default"
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
    basename(abspath(path.module)) = var.name
  }

  squad_label = {
    "squad" : var.squad
    "team" : var.squad
  }
  product_label = var.product != "" ? {
    product = var.product
  } : {}
}

resource "google_container_cluster" "this" {
  name    = var.name
  project = var.project

  network    = var.network
  subnetwork = var.subnetwork

  location       = var.region
  node_locations = var.zones

  deletion_protection = var.deletion_protection

  enable_autopilot = true

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

  # https://github.com/hashicorp/terraform-provider-google/issues/9505#issuecomment-1340074019
  cluster_autoscaling {
    auto_provisioning_defaults {
      service_account = google_service_account.cluster_default.email
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

  timeouts {
    create = "30m"
    update = "30m"
    delete = "30m"
  }

  resource_labels = merge(local.default_labels, local.squad_label, local.product_label)

  lifecycle {
    # https://github.com/hashicorp/terraform-provider-google/issues/6901
    ignore_changes = [initial_node_count]
  }

  depends_on = [google_service_account.cluster_default]
}
