// Create a global network in which to place our resources.
// checkov:skip=CKV2_GCP_18: firewalls are owned by consumers of this network (e.g. the gke module), not this base module.
resource "google_compute_network" "this" {
  name                            = var.name
  auto_create_subnetworks         = false
  routing_mode                    = "GLOBAL"
  project                         = var.project_id
  delete_default_routes_on_create = true
}

// Create a default route to the Internet.
resource "google_compute_route" "egress-inet" {
  name    = var.name
  network = google_compute_network.this.name

  dest_range       = "0.0.0.0/0"
  next_hop_gateway = "default-internet-gateway"
}

// Create regional subnets in each of the specified regions,
// which we will use to operate Cloud Run services.
resource "google_compute_subnetwork" "regional" {
  for_each = {
    for region in var.regions : region => index(var.regions, region)
  }

  name = "${var.name}-${each.key}"

  // This is needed in order to interact with Google APIs like Pub/Sub.
  private_ip_google_access = true

  network       = google_compute_network.this.id
  region        = each.key
  ip_cidr_range = cidrsubnet(var.cidr, 8, var.netnum_offset + each.value)

  log_config {
    aggregation_interval = "INTERVAL_15_MIN"
    flow_sampling        = 1.0
    metadata             = "INCLUDE_ALL_METADATA"
  }
}

// Cloud DNS query logging for private zones is configured via a DNS Server
// Policy attached to the network, not via cloud_logging_config on the zone
// (which GCP rejects on private zones). See
// https://cloud.google.com/dns/docs/monitoring#enabling_and_disabling_logging_for_private_managed_zones
resource "google_dns_policy" "dns_logging_policy" {
  name           = var.name
  enable_logging = var.hosted_zone_logging_enabled
  networks {
    network_url = google_compute_network.this.id
  }
}
