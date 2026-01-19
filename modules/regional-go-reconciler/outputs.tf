/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

output "receiver" {
  description = "The workqueue receiver object for connecting triggers."
  value       = var.shards == 1 ? module.workqueue[0].receiver : module.workqueue-sharded[0].receiver
}

output "reconciler-uris" {
  description = "The URIs of the reconciler service by region."
  value       = module.reconciler.uris
}
