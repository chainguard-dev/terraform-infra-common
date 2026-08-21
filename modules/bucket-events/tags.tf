// Copyright 2026 Chainguard, Inc.
// SPDX-License-Identifier: Apache-2.0

// The binding parent needs the numeric project number. Read only when tags are set.
data "google_project" "resource_manager_tags" {
  count = length(var.resource_manager_tags) > 0 ? 1 : 0

  project_id = var.project_id
}

// Pub/Sub bindings are global. Not google_pubsub_topic.tags: that field is
// immutable and replaces the topic.
resource "google_tags_tag_binding" "dead_letter_topic" {
  for_each = var.resource_manager_tags

  parent    = "//pubsub.googleapis.com/projects/${data.google_project.resource_manager_tags[0].number}/topics/${google_pubsub_topic.dead-letter.name}"
  tag_value = each.value
}

resource "google_tags_tag_binding" "internal_topic" {
  for_each = var.resource_manager_tags

  parent    = "//pubsub.googleapis.com/projects/${data.google_project.resource_manager_tags[0].number}/topics/${google_pubsub_topic.internal.name}"
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
