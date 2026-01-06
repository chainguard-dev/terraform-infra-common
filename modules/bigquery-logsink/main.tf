terraform {
  required_providers {
    google = {
      source = "hashicorp/google"
    }
  }
}

locals {
  default_labels = {
    basename(abspath(path.module)) = var.name
    terraform-module               = basename(abspath(path.module))
  }

  merged_labels = merge(
    local.default_labels,
    var.team != null ? { team = var.team } : {},
    var.product != null ? { product = var.product } : {},
    var.labels
  )
}

resource "google_bigquery_dataset" "this" {
  project    = var.project_id
  dataset_id = "log_sinks_${replace(var.name, "-", "_")}"
  location   = var.location

  description = var.dataset_description

  delete_contents_on_destroy = var.delete_contents_on_destroy

  labels = local.merged_labels
}
