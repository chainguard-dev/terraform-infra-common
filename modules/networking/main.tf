// Create a global network in which to place our resources.
resource "google_compute_network" "this" {
  // Only create network if one was not explicitly passed in.
  count                           = var.network != null ? 0 : 1
  name                            = var.name
  auto_create_subnetworks         = false
  routing_mode                    = "GLOBAL"
  project                         = var.project_id
  delete_default_routes_on_create = true
}

locals {
  network = var.network != null ? var.network : google_compute_network.this[0].id
}

// Create a default route to the Internet.
resource "google_compute_route" "egress-inet" {
  name    = var.name
  network = local.network

  tags             = ["egress-inet"]
  dest_range       = "0.0.0.0/0"
  next_hop_gateway = "default-internet-gateway"
}

// Create a route to googleapis.com to access GCP APIs.
// See https://cloud.google.com/vpc/docs/configure-private-google-access#config-options for more details.
resource "google_compute_route" "private-googleapis" {
  for_each = toset(["199.36.153.8/30", "34.126.0.0/18"])
  name     = var.name
  network  = local.network

  dest_range       = each.value
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

  network       = local.network
  region        = each.key
  ip_cidr_range = cidrsubnet(var.cidr, 8, var.netnum_offset + each.value)
}
