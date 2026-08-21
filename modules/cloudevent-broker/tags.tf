// Copyright 2026 Chainguard, Inc.
// SPDX-License-Identifier: Apache-2.0

// The binding parent needs the numeric project number. Read only when tags are set.
data "google_project" "resource_manager_tags" {
  count = length(var.resource_manager_tags) > 0 ? 1 : 0

  project_id = var.project_id
}

// Pub/Sub bindings are global. Not google_pubsub_topic.tags: that field is
// immutable and replaces the topic.
locals {
  regional_topic_tag_bindings = {
    for pair in setproduct(keys(google_pubsub_topic.this), keys(var.resource_manager_tags)) :
    "${pair[0]}/${pair[1]}" => {
      topic     = google_pubsub_topic.this[pair[0]].name
      tag_value = var.resource_manager_tags[pair[1]]
    }
  }

  dedicated_topic_tag_bindings = {
    for pair in setproduct(keys(google_pubsub_topic.dedicated), keys(var.resource_manager_tags)) :
    "${pair[0]}/${pair[1]}" => {
      topic     = google_pubsub_topic.dedicated[pair[0]].name
      tag_value = var.resource_manager_tags[pair[1]]
    }
  }
}

resource "google_tags_tag_binding" "topic" {
  for_each = local.regional_topic_tag_bindings

  parent    = "//pubsub.googleapis.com/projects/${data.google_project.resource_manager_tags[0].number}/topics/${each.value.topic}"
  tag_value = each.value.tag_value
}

resource "google_tags_tag_binding" "dedicated_topic" {
  for_each = local.dedicated_topic_tag_bindings

  parent    = "//pubsub.googleapis.com/projects/${data.google_project.resource_manager_tags[0].number}/topics/${each.value.topic}"
  tag_value = each.value.tag_value
}
