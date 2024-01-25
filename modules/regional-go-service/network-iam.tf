// This file gramnts permissions for Cloud Run to access VPCs.
// In particular, this is configured to support connecting to a shared VPC in another project,
// but should be safe to apply in the same project.
// See https://cloud.google.com/run/docs/configuring/shared-vpc-direct-vpc#set_up_iam_permissions for more details.

data "google_project" "project" {
  project_id = var.project_id
}

// Grant Cloud Run the service access to the network.
resource "google_project_iam_member" "cloudrun_service_network_user" {
  project = var.network_project
  member  = "serviceAccount:service-${data.google_project.project.number}@serverless-robot-prod.iam.gserviceaccount.com"
  role    = "roles/compute.networkUser"
}

// Grant service account access to see networks on the network project.
resource "google_project_iam_member" "project_network_viewer" {
  project = var.network_project
  member  = "serviceAccount:${var.service_account}"
  role    = "roles/compute.networkViewer"
}

// Grant service account access to use subnet. This is typically granted with roles/run.serviceAgent,
// but that role does not necessarily grant access if the network resides in another project.
// See https://cloud.google.com/run/docs/configuring/vpc-direct-vpc#direct-vpc-service for more details.
resource "google_compute_subnetwork_iam_member" "subnet_network_user" {
  for_each = var.regions

  // If not set, provider project should be used.
  project    = var.network_project
  region     = each.key
  subnetwork = each.value.subnet
  role       = "roles/compute.networkUser"
  member     = "serviceAccount:${var.service_account}"
}

