# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

# for_each keys of the burn-rate alert policies actually created, by scope.
# Exists so tests/ can assert at plan time which policies the alerting knobs
# produce (policy IDs are unknown until apply; keys are known at plan).
output "alert_policy_keys" {
  description = "Keys of the created burn-rate alert policies: rolling-period keys for multi_region/gclb, region-period keys for per_region."
  value = {
    multi_region = keys(google_monitoring_alert_policy.slo_burn_rate_multi_region)
    per_region   = keys(google_monitoring_alert_policy.slo_burn_rate_per_region)
    gclb         = keys(google_monitoring_alert_policy.slo_burn_rate_gclb)
  }
}
