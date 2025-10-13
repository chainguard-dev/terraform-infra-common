terraform {
  required_providers {
    ko     = { source = "ko-build/ko" }
    cosign = { source = "chainguard-dev/cosign" }
  }
}

data "google_project" "project" {
  project_id = var.project_id
}

locals {
  regional-types = merge([
    for region in keys(var.regions) : merge([
      for type in keys(var.types) : {
        "${region}-${type}" : {
          region = region
          type   = type
        }
      }
    ]...)
  ]...)

  default_labels = {
    basename(abspath(path.module)) = var.name
    terraform-module               = basename(abspath(path.module))
  }

  effective_team = coalesce(var.team, var.squad, "unknown")

  squad_label = {
    squad = local.effective_team
    team  = local.effective_team
  }
  product_label = var.product != "" ? {
    product = var.product
  } : {}

  merged_labels = merge(local.default_labels, local.squad_label, local.product_label, var.labels)
}

resource "random_id" "suffix" {
  byte_length = 2
}

resource "google_storage_bucket" "recorder" {
  for_each = var.regions

  name          = "${var.name}-${each.key}-${random_id.suffix.hex}"
  project       = var.project_id
  location      = each.key
  force_destroy = !var.deletion_protection
  labels        = local.merged_labels

  uniform_bucket_level_access = true

  lifecycle_rule {
    action {
      type = "Delete"
    }
    condition {
      // 1 week + 1 day buffer
      age = 8
    }
  }
}

// What identity is deploying this?
data "google_client_openid_userinfo" "me" {}
