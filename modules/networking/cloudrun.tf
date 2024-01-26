// This file grants the Cloud Run service access to shared network resources, even if they exist in a different project.

data "google_project" "cloudrun_service_project" {
  // Prefer service_project_id, but fall back to project_id.
  // Technically we don't need this if the service_project_id == project_id,
  // but it's safe to always do this, since we'd be granting permissions the service account
  // already has.
  project_id = var.service_project_id != null ? var.service_project_id : var.project_id
}

// Grant Cloud Run access to view networks.
// See https://cloud.google.com/run/docs/configuring/shared-vpc-direct-vpc#set_up_iam_permissions
resource "google_project_iam_member" "cloudrun_service_network_user" {
  project = var.project_id
  member  = "serviceAccount:service-${data.google_project.cloudrun_service_project.number}@serverless-robot-prod.iam.gserviceaccount.com"
  role    = "roles/compute.networkViewer"
}

// Grant Cloud Run the service access to use the networks.
// See https://cloud.google.com/run/docs/configuring/shared-vpc-direct-vpc#set_up_iam_permissions
resource "google_compute_subnetwork_iam_member" "subnet_network_user" {
  for_each = resource.google_compute_subnetwork.regional

  // If not set, provider project should be used.
  project    = var.project_id
  region     = each.value.region
  subnetwork = each.value.name
  role       = "roles/compute.networkUser"
  member     = "serviceAccount:service-${data.google_project.cloudrun_service_project.number}@serverless-robot-prod.iam.gserviceaccount.com"
}
