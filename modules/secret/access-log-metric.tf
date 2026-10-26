# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

resource "google_logging_metric" "anomalous_secret_access" {
  count       = var.access_log_metric_name == null ? 0 : 1
  project     = var.project_id
  name        = var.access_log_metric_name
  description = "Unauthorized access to secret ${var.name}; uses the local alert's identity exemptions."
  filter      = local.alert_filter

  metric_descriptor {
    metric_kind = "DELTA"
    value_type  = "INT64"
    unit        = "1"
  }
}

output "access_log_metric_name" {
  description = "Unauthorized-access log counter name, or null when disabled. Alert policies belong in the caller's monitoring project."
  value       = one(google_logging_metric.anomalous_secret_access[*].name)
}
