# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

output "configuration" {
  sensitive = true
  value     = module.load_balancer.configuration
}

output "records" {
  value = module.domain.records
}
