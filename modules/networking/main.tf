// Create a global network in which to place our resources.
resource "google_compute_network" "this" {
  name                            = var.name
  auto_create_subnetworks         = false
  routing_mode                    = "GLOBAL"
  project                         = var.project_id
  delete_default_routes_on_create = true
}

// Create the global address that will be used to peer the network.
// See https://cloud.google.com/vpc/docs/configure-private-services-access#terraform
resource "google_compute_global_address" "nw-peer" {
  count = var.create_servicenetworking_peer ? 1 : 0

  name = "${var.name}-peer"
  // If not set, provider project should be used.
  project = var.project_id
  network = google_compute_network.this.id

  address_type  = "INTERNAL"
  purpose       = "VPC_PEERING"
  prefix_length = 16
}

// Create the peering connection.
// See https://cloud.google.com/vpc/docs/configure-private-services-access#terraform
resource "google_service_networking_connection" "nw-peer-connect" {
  count = var.create_servicenetworking_peer ? 1 : 0

  network                 = google_compute_network.this.id
  service                 = "servicenetworking.googleapis.com"
  reserved_peering_ranges = [google_compute_global_address.nw-peer.name]
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
}
