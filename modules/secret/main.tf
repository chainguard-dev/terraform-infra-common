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

// What identity is deploying this?
data "google_client_openid_userinfo" "me" {}

// Create an alert policy to notify if the secret is accessed by an unauthorized entity.
resource "google_monitoring_alert_policy" "anomalous-secret-access" {
  # In the absence of data, incident will auto-close after an hour
  alert_strategy {
    auto_close = "3600s"

    notification_rate_limit {
      period = "3600s" // re-alert hourly if condition still valid.
    }
  }

  display_name = "Abnormal Secret Access: ${var.name}"
  combiner     = "OR"

  conditions {
    display_name = "Abnormal Secret Access: ${var.name}"

    condition_matched_log {
      filter = <<EOT
      protoPayload.serviceName="secretmanager.googleapis.com"
      (
        protoPayload.request.name: ("projects/${var.project_id}/secrets/${var.name}/" OR "projects/${data.google_project.project.number}/secrets/${var.name}/") OR
        protoPayload.request.parent=("projects/${var.project_id}/secrets/${var.name}" OR "projects/${data.google_project.project.number}/secrets/${var.name}")
      )

      -- Ignore the identity that is intended to access this.
      -(
        protoPayload.authenticationInfo.principalEmail="${var.service-account}"
        protoPayload.methodName="google.cloud.secretmanager.v1.SecretManagerService.AccessSecretVersion"
      )
      EOT
    }
  }

  notification_channels = var.notification-channels

  enabled = "true"
  project = var.project_id
}
