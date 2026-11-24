# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0
#
# Run in CI by .chainguard/ci/tf-module-tests.yaml.

mock_provider "google" {}

variables {
  project_id        = "fixture-project"
  name              = "fixture"
  enable_monitoring = true
  sinks = {
    tracking = {
      log_filter = "jsonPayload.log_type=\"tracking\""
    }
  }
}

# A low-volume sink exports a few entries per 5-minute window. The threshold
# must compare entries per window, not entries per second, or a working sink
# stays below 1 and the alert never closes.
run "condition_counts_entries_per_window" {
  command = plan

  assert {
    condition     = google_monitoring_alert_policy.log_ingestion["tracking"].conditions[0].condition_threshold[0].aggregations[0].per_series_aligner == "ALIGN_SUM"
    error_message = "the aligner must sum entries per window; ALIGN_RATE is per second"
  }
  assert {
    condition     = google_monitoring_alert_policy.log_ingestion["tracking"].conditions[0].condition_threshold[0].aggregations[0].alignment_period == "300s"
    error_message = "the alignment period must stay 300s"
  }
  assert {
    condition     = google_monitoring_alert_policy.log_ingestion["tracking"].conditions[0].condition_threshold[0].comparison == "COMPARISON_LT"
    error_message = "the condition must fire below the threshold"
  }
  assert {
    condition     = google_monitoring_alert_policy.log_ingestion["tracking"].conditions[0].condition_threshold[0].threshold_value == 1
    error_message = "the threshold must be one entry per window"
  }
  assert {
    condition     = google_monitoring_alert_policy.log_ingestion["tracking"].conditions[0].condition_threshold[0].duration == "86400s"
    error_message = "the duration must follow alert_threshold_minutes (default 1440)"
  }
}

# A sink that exports nothing writes no points, so only missing-data handling
# can open the incident.
run "missing_data_counts_as_met" {
  command = plan

  assert {
    condition     = google_monitoring_alert_policy.log_ingestion["tracking"].conditions[0].condition_threshold[0].evaluation_missing_data == "EVALUATION_MISSING_DATA_ACTIVE"
    error_message = "a sink that stops exporting must open the incident"
  }
}

run "monitoring_disabled_creates_no_policies" {
  command = plan

  variables {
    enable_monitoring = false
  }

  assert {
    condition     = length(google_monitoring_alert_policy.log_ingestion) == 0
    error_message = "enable_monitoring = false must create no alert policies"
  }
}
