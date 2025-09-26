/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

output "receiver" {
  description = "The workqueue receiver object for connecting triggers."
  value       = module.workqueue.receiver
}

output "reconciler-uris" {
  description = "The URIs of the reconciler service by region."
  value       = module.reconciler.uris
}
