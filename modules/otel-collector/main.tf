terraform {
  required_providers {
    ko     = { source = "ko-build/ko" }
    cosign = { source = "chainguard-dev/cosign" }
  }
}

resource "google_project_iam_member" "metrics-writer" {
  project = var.project_id
  role    = "roles/monitoring.metricWriter"
  member  = "serviceAccount:${var.service_account}"
}

resource "google_project_iam_member" "trace-writer" {
  project = var.project_id
  role    = "roles/cloudtrace.agent"
  member  = "serviceAccount:${var.service_account}"
}

resource "ko_build" "otel-image" {
  base_image  = var.otel_collector_image
  importpath  = "./cmd/otel-collector"
  working_dir = path.module
}

resource "cosign_sign" "otel-image" {
  image = ko_build.otel-image.image_ref

  # Only keep the latest signature.
  conflict = "REPLACE"
}
