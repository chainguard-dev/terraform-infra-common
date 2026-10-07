/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

output "alert_policy_name" {
  description = "The resource name of the Service Health alert policy (projects/<project>/alertPolicies/<id>)."
  value       = google_monitoring_alert_policy.incident.name
}

output "filter" {
  description = "The Cloud Logging filter matched by the alert policy, for reuse in log views or sinks."
  value       = local.filter
}
