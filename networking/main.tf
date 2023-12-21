// Create a global network in which to place our resources.
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
    for region in var.regions : region => 1 + index(var.regions, region)
  }

  name = "${var.name}-${each.key}"

  // This is needed in order to interact with Google APIs like Pub/Sub.
  private_ip_google_access = true

  network       = google_compute_network.this.id
  region        = each.key
  ip_cidr_range = cidrsubnet(var.cidr, 8, each.value)
}
