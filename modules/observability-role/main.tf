/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// A single role carrying the union of the permissions of
// roles/monitoring.metricWriter, roles/cloudtrace.agent, and
// roles/cloudprofiler.agent. Granting this one role to a service account
// (via the observability_role variable of the service modules) costs one
// project IAM policy member entry instead of three, which keeps projects
// with many services under the policy's hard limit of 1,500 members.
resource "google_project_iam_custom_role" "this" {
  project     = var.project_id
  role_id     = var.role_id
  title       = var.title
  description = "Union of monitoring.metricWriter, cloudtrace.agent, and cloudprofiler.agent for Cloud Run service observability."
  permissions = [
    // roles/monitoring.metricWriter
    "monitoring.metricDescriptors.create",
    "monitoring.metricDescriptors.get",
    "monitoring.metricDescriptors.list",
    "monitoring.monitoredResourceDescriptors.get",
    "monitoring.monitoredResourceDescriptors.list",
    "monitoring.timeSeries.create",
    // roles/cloudtrace.agent
    "cloudtrace.traces.patch",
    "telemetry.traces.write",
    // roles/cloudprofiler.agent
    "cloudprofiler.profiles.create",
    "cloudprofiler.profiles.update",
  ]
}
