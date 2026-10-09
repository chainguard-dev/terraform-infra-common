# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

mock_provider "google" {}
mock_provider "random" {}

variables {
  project_id = "fixture-project"
  name       = "maintenance"
  team       = "platform"
  identity   = "serviceAccount:worker@fixture-project.iam.gserviceaccount.com"
  regions = {
    us-central1 = { network = "unused", subnet = "unused" }
    us-east1    = {}
  }
}

run "regional_buckets_and_single_identity" {
  command = plan

  assert {
    condition     = length(google_storage_bucket.leases) == 2 && length(google_storage_bucket_iam_member.participant) == 2
    error_message = "Create exactly one bucket and one identity grant per region."
  }
  assert {
    condition     = alltrue([for region, b in google_storage_bucket.leases : lower(b.location) == region && b.project == var.project_id && b.storage_class == "STANDARD"])
    error_message = "Buckets must reside in their requested regions and project."
  }
  assert {
    condition     = alltrue([for b in google_storage_bucket.leases : b.uniform_bucket_level_access && b.public_access_prevention == "enforced" && !b.force_destroy && !b.versioning[0].enabled && b.soft_delete_policy[0].retention_duration_seconds == 0])
    error_message = "Lease buckets must be private, protected from destruction, and must not retain overwritten lease versions."
  }
  assert {
    condition     = alltrue([for g in google_storage_bucket_iam_member.participant : g.member == var.identity && g.role == "roles/storage.objectUser"])
    error_message = "Only the supplied identity gets object access; bucket administration is not granted."
  }
}

run "reject_public_identity" {
  command = plan
  variables { identity = "allUsers" }
  expect_failures = [var.identity]
}

run "require_regions" {
  command = plan
  variables { regions = {} }
  expect_failures = [var.regions]
}

run "team_and_product_are_not_labels" {
  command = plan
  variables { product = "containers" }
  assert {
    condition     = alltrue([for b in google_storage_bucket.leases : !contains(keys(b.labels), "team") && !contains(keys(b.labels), "product")])
    error_message = "Buckets support tags, so team and product must not become labels."
  }
}

run "caller_labels_override_defaults" {
  command = plan
  variables {
    product = "containers"
    labels  = { team = "custom", product = "override", terraform-module = "custom-module" }
  }
  assert {
    condition     = alltrue([for b in google_storage_bucket.leases : b.labels["team"] == "custom" && b.labels["product"] == "override" && b.labels["terraform-module"] == "custom-module"])
    error_message = "Caller labels must take precedence, matching regional-go-cron."
  }
}
