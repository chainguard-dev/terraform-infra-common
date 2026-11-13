# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

// Keep the initial provider-assigned IDs to distinguish updates from replacements.
resource "terraform_data" "original_pool_ids" {
  for_each = var.pools
  input    = google_container_node_pool.pools[each.key].id
  lifecycle {
    ignore_changes = [input]
  }
}
