/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

output "internal_ip" {
  description = "Internal IP address of the bastion VM"
  value       = google_compute_instance.bastion.network_interface[0].network_ip
}

output "instance_name" {
  description = "Name of the bastion compute instance"
  value       = google_compute_instance.bastion.name
}

output "service_account_email" {
  description = "Service account email used by the bastion"
  value       = google_service_account.bastion_sa.email
}

output "ssh_target_tag" {
  description = "Network tag applied to the bastion for SSH firewall rules"
  value       = local.instance_tag
}

output "nat_router_name" {
  description = "Name of the Cloud NAT router (empty when enable_nat = false)"
  value       = var.enable_nat ? google_compute_router_nat.bastion_nat[0].name : ""
}
