# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0
#
# The scaling-failure and failed-request policies carry operator guidance
# whose Logs Explorer links are built from the same filter the condition
# matches on, so a mocked plan shows the two stay in step.

mock_provider "google" {}

variables {
  project_id = "fixture-project"
  team       = "fixture"
}

run "scaling_policies_document_what_fired_and_link_the_logs" {
  command = plan

  assert {
    condition     = strcontains(google_monitoring_alert_policy.cloud-run-scaling-failure[0].documentation[0].content, "What to check")
    error_message = "the scaling-failure policy must carry troubleshooting steps."
  }
  assert {
    condition     = google_monitoring_alert_policy.cloud-run-scaling-failure[0].documentation[0].links[0].url == "https://console.cloud.google.com/logs/query;query=${urlencode(google_monitoring_alert_policy.cloud-run-scaling-failure[0].conditions[0].condition_matched_log[0].filter)}?project=fixture-project"
    error_message = "the scaling-failure Logs Explorer link must open the filter the condition matches on."
  }
  assert {
    condition     = strcontains(google_monitoring_alert_policy.cloud-run-failed-req.documentation[0].content, "What to check")
    error_message = "the failed-request policy must carry troubleshooting steps."
  }
  assert {
    condition     = google_monitoring_alert_policy.cloud-run-failed-req.documentation[0].links[0].url == "https://console.cloud.google.com/logs/query;query=${urlencode(google_monitoring_alert_policy.cloud-run-failed-req.conditions[0].condition_matched_log[0].filter)}?project=fixture-project"
    error_message = "the failed-request Logs Explorer link must open the filter the condition matches on."
  }
  assert {
    condition     = strcontains(google_monitoring_alert_policy.cloud-run-failed-req.documentation[0].links[1].url, urlencode("NOT log_id(\"run.googleapis.com/requests\")"))
    error_message = "the failed-request error-logs link must leave out the request log."
  }
}
