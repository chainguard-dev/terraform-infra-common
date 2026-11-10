# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

output "tag_bindings" {
  description = "Cloud Run tag binding parents, values and locations, keyed by region/tag key."
  value       = module.this.tag_bindings
}
