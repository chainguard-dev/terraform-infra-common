output "network_id" {
  value = google_compute_network.this.id
}

output "regional-networks" {
  value = {
    for region in var.regions : region => {
      network = google_compute_network.this.id
      subnet  = google_compute_subnetwork.regional[region].name
    }
  }
}
