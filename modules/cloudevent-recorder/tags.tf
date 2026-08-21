// Copyright 2026 Chainguard, Inc.
// SPDX-License-Identifier: Apache-2.0

locals {
  resource_manager_tag_bindings = {
    for pair in setproduct(keys(var.regions), keys(var.resource_manager_tags)) :
    "${pair[0]}/${pair[1]}" => {
      location  = pair[0]
      tag_value = var.resource_manager_tags[pair[1]]
    }
  }
}

resource "google_tags_location_tag_binding" "recorder_bucket" {
  for_each = local.resource_manager_tag_bindings

  parent    = "//storage.googleapis.com/projects/_/buckets/${google_storage_bucket.recorder[each.value.location].name}"
  tag_value = each.value.tag_value
  location  = lower(google_storage_bucket.recorder[each.value.location].location)
}

// Pub/Sub bindings are global. Not google_pubsub_topic.tags: that field is
// immutable and replaces the topic.
locals {
  pubsub_topic_tag_bindings = {
    for pair in setproduct(keys(google_pubsub_topic.dead-letter), keys(var.resource_manager_tags)) :
    "${pair[0]}/${pair[1]}" => {
      name      = google_pubsub_topic.dead-letter[pair[0]].name
      tag_value = var.resource_manager_tags[pair[1]]
    }
  }

  dlq_subscription_tag_bindings = {
    for pair in setproduct(keys(google_pubsub_subscription.dead-letter-pull-sub), keys(var.resource_manager_tags)) :
    "${pair[0]}/${pair[1]}" => {
      name      = google_pubsub_subscription.dead-letter-pull-sub[pair[0]].name
      tag_value = var.resource_manager_tags[pair[1]]
    }
  }

  subscription_tag_bindings = {
    for pair in setproduct(keys(google_pubsub_subscription.this), keys(var.resource_manager_tags)) :
    "${pair[0]}/${pair[1]}" => {
      name      = google_pubsub_subscription.this[pair[0]].name
      tag_value = var.resource_manager_tags[pair[1]]
    }
  }

  bigquery_table_tag_bindings = {
    for pair in setproduct(keys(google_bigquery_table.types), keys(var.resource_manager_tags)) :
    "${pair[0]}/${pair[1]}" => {
      table_id  = google_bigquery_table.types[pair[0]].table_id
      tag_value = var.resource_manager_tags[pair[1]]
    }
  }
}

resource "google_tags_tag_binding" "dead_letter_topic" {
  for_each = local.pubsub_topic_tag_bindings

  parent    = "//pubsub.googleapis.com/projects/${data.google_project.project.number}/topics/${each.value.name}"
  tag_value = each.value.tag_value
}

resource "google_tags_tag_binding" "dead_letter_subscription" {
  for_each = local.dlq_subscription_tag_bindings

  parent    = "//pubsub.googleapis.com/projects/${data.google_project.project.number}/subscriptions/${each.value.name}"
  tag_value = each.value.tag_value
}

resource "google_tags_tag_binding" "subscription" {
  for_each = local.subscription_tag_bindings

  parent    = "//pubsub.googleapis.com/projects/${data.google_project.project.number}/subscriptions/${each.value.name}"
  tag_value = each.value.tag_value
}

// BigQuery tag bindings are location scoped to the dataset location. Note that
// hashicorp/terraform-provider-google#18254 is open: the provider builds the
// wrong request URL for multi-region locations (US, EU), so these bindings only
// apply cleanly when var.location is a single region.
resource "google_tags_location_tag_binding" "bigquery_dataset" {
  for_each = var.resource_manager_tags

  parent    = "//bigquery.googleapis.com/projects/${data.google_project.project.number}/datasets/${google_bigquery_dataset.this.dataset_id}"
  tag_value = each.value
  location  = lower(var.location)
}

resource "google_tags_location_tag_binding" "bigquery_table" {
  for_each = local.bigquery_table_tag_bindings

  parent    = "//bigquery.googleapis.com/projects/${data.google_project.project.number}/datasets/${google_bigquery_dataset.this.dataset_id}/tables/${each.value.table_id}"
  tag_value = each.value.tag_value
  location  = lower(var.location)
}
