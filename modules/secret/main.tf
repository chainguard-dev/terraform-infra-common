// Create the GCP secret to hold the configuration data.
resource "google_secret_manager_secret" "this" {
  secret_id = var.name
  replication {
    auto {}
  }
  lifecycle {
    prevent_destroy = true
  }
}

// Create a placeholder GCP secret version to avoid bad reference on first deploy.
resource "google_secret_manager_secret_version" "placeholder" {
  count = var.create_placeholder_version ? 1 : 0

  secret      = google_secret_manager_secret.this.id
  secret_data = "placeholder"

  lifecycle {
    prevent_destroy = true
  }
}

// Only the service account as which the service runs should have access to the secret.
resource "google_secret_manager_secret_iam_binding" "authorize-service-access" {
  secret_id = google_secret_manager_secret.this.id
  role      = "roles/secretmanager.secretAccessor"
  members   = ["serviceAccount:${var.service-account}"]
}

// Authorize the specified identity to add new secret values.
resource "google_secret_manager_secret_iam_binding" "authorize-version-adder" {
  project   = var.project_id
  secret_id = google_secret_manager_secret.this.secret_id
  role      = "roles/secretmanager.secretVersionAdder"
  members   = [var.authorized-adder]
}

// Get a project number for this project ID.
data "google_project" "project" { project_id = var.project_id }

