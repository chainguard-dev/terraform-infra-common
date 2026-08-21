// Copyright 2026 Chainguard, Inc.
// SPDX-License-Identifier: Apache-2.0

// The binding parent needs the numeric project number. Read only when tags are set.
data "google_project" "resource_manager_tags" {
  count = length(var.resource_manager_tags) > 0 ? 1 : 0

  project_id = var.project_id
}

// The DLQ bucket is optional, so bind nothing when it is disabled.
resource "google_tags_location_tag_binding" "dlq_bucket" {
  // Mirror the bucket's own count condition exactly. `enable_dlq_bucket` is
  // nullable, and the bucket treats null as enabled via `== false`, whereas a
  // bare `var.enable_dlq_bucket ?` would fail with "condition value is null".
  for_each = var.enable_dlq_bucket == false ? {} : var.resource_manager_tags

  parent    = "//storage.googleapis.com/projects/_/buckets/${google_storage_bucket.dlq_bucket[0].name}"
  tag_value = each.value
  location  = lower(google_storage_bucket.dlq_bucket[0].location)
}

// Pub/Sub bindings are global. Not google_pubsub_topic.tags: that field is
// immutable and replaces the topic.
resource "google_tags_tag_binding" "dead_letter_topic" {
  for_each = var.resource_manager_tags

  parent    = "//pubsub.googleapis.com/projects/${data.google_project.resource_manager_tags[0].number}/topics/${google_pubsub_topic.dead-letter.name}"
  tag_value = each.value
}

resource "google_tags_tag_binding" "dead_letter_subscription" {
  for_each = var.resource_manager_tags

  parent    = "//pubsub.googleapis.com/projects/${data.google_project.resource_manager_tags[0].number}/subscriptions/${google_pubsub_subscription.dead-letter-pull-sub.name}"
  tag_value = each.value
}

resource "google_tags_tag_binding" "subscription" {
  for_each = var.resource_manager_tags

  parent    = "//pubsub.googleapis.com/projects/${data.google_project.resource_manager_tags[0].number}/subscriptions/${google_pubsub_subscription.this.name}"
  tag_value = each.value
}
