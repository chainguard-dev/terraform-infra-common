output "network_id" {
  value = local.network
}

output "regional-networks" {
  value = {
    for region in var.regions : region => {
      network = local.network
      subnet  = google_compute_subnetwork.regional[region].name
    }
  }
}
