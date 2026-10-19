# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0
#
# The http error rate policy's filters, condition names and documentation are
# rendered from the module inputs, so a mocked plan shows the text Cloud
# Monitoring receives.

mock_provider "google" {}

variables {
  project_id = "fixture-project"
  team       = "fixture"
}

run "several_methods_render_one_regex" {
  command = plan

  variables {
    http_error_method_status_exclusions = {
      "svc-a|svc-b" = { methods = ["GET", "HEAD"], codes = ["401", "403", "404", "409"] }
    }
  }

  assert {
    condition     = strcontains(google_monitoring_alert_policy.http_error_rate[0].conditions[0].condition_threshold[0].filter, "metric.labels.service_name != monitoring.regex.full_match(\"svc-a|svc-b\")")
    error_message = "the team-wide condition must leave out the services that have exclusions."
  }
  assert {
    condition     = strcontains(google_monitoring_alert_policy.http_error_rate[0].conditions[1].condition_threshold[0].filter, "metric.labels.method != monitoring.regex.full_match(\"GET|HEAD\")")
    error_message = "the first replacement condition must count methods other than GET and HEAD."
  }
  assert {
    condition     = google_monitoring_alert_policy.http_error_rate[0].conditions[1].display_name == "svc-a|svc-b http error rate (non-(GET|HEAD))"
    error_message = "unexpected first replacement condition name: ${google_monitoring_alert_policy.http_error_rate[0].conditions[1].display_name}"
  }
  assert {
    condition     = strcontains(google_monitoring_alert_policy.http_error_rate[0].conditions[2].condition_threshold[0].filter, "metric.labels.method = monitoring.regex.full_match(\"GET|HEAD\")")
    error_message = "the second replacement condition must count GET and HEAD."
  }
  assert {
    condition     = strcontains(google_monitoring_alert_policy.http_error_rate[0].conditions[2].condition_threshold[0].filter, "metric.labels.code != monitoring.regex.full_match(\"401|403|404|409\")")
    error_message = "the second replacement condition must exempt the listed codes."
  }
  assert {
    condition     = google_monitoring_alert_policy.http_error_rate[0].conditions[2].display_name == "svc-a|svc-b http error rate (GET|HEAD, non-401,403,404,409)"
    error_message = "unexpected second replacement condition name: ${google_monitoring_alert_policy.http_error_rate[0].conditions[2].display_name}"
  }
  # REDUCE_MEAN averages the series a group matches, so GET and HEAD sharing a
  # group would dilute a GET error rate with HEAD series.
  assert {
    condition     = join(",", google_monitoring_alert_policy.http_error_rate[0].conditions[2].condition_threshold[0].aggregations[0].group_by_fields) == "metric.label.\"team\",resource.label.\"job\",metric.label.\"method\""
    error_message = "the matched methods must be averaged separately: ${join(",", google_monitoring_alert_policy.http_error_rate[0].conditions[2].condition_threshold[0].aggregations[0].group_by_fields)}"
  }
  assert {
    condition     = join(",", google_monitoring_alert_policy.http_error_rate[0].conditions[1].condition_threshold[0].aggregations[0].group_by_fields) == "metric.label.\"team\",resource.label.\"job\""
    error_message = "the condition for other methods must keep its grouping: ${join(",", google_monitoring_alert_policy.http_error_rate[0].conditions[1].condition_threshold[0].aggregations[0].group_by_fields)}"
  }
}

run "single_method_renders_literal" {
  command = plan

  variables {
    http_error_method_status_exclusions = {
      "grype" = { method = "POST", codes = ["409"] }
    }
  }

  assert {
    condition     = strcontains(google_monitoring_alert_policy.http_error_rate[0].conditions[1].condition_threshold[0].filter, "metric.labels.method != \"POST\"")
    error_message = "a single method must keep its literal comparison."
  }
  assert {
    condition     = strcontains(google_monitoring_alert_policy.http_error_rate[0].conditions[2].condition_threshold[0].filter, "metric.labels.method = \"POST\"")
    error_message = "a single method must keep its literal comparison."
  }
  assert {
    condition     = google_monitoring_alert_policy.http_error_rate[0].conditions[1].display_name == "grype http error rate (non-POST)"
    error_message = "single-method condition name changed: ${google_monitoring_alert_policy.http_error_rate[0].conditions[1].display_name}"
  }
  assert {
    condition     = google_monitoring_alert_policy.http_error_rate[0].conditions[2].display_name == "grype http error rate (POST, non-409)"
    error_message = "single-method condition name changed: ${google_monitoring_alert_policy.http_error_rate[0].conditions[2].display_name}"
  }
  assert {
    condition     = join(",", google_monitoring_alert_policy.http_error_rate[0].conditions[2].condition_threshold[0].aggregations[0].group_by_fields) == "metric.label.\"team\",resource.label.\"job\""
    error_message = "a single method must keep its grouping: ${join(",", google_monitoring_alert_policy.http_error_rate[0].conditions[2].condition_threshold[0].aggregations[0].group_by_fields)}"
  }
}

run "one_entry_methods_list_renders_literal" {
  command = plan

  variables {
    http_error_method_status_exclusions = {
      "grype" = { methods = ["POST"], codes = ["409"] }
    }
  }

  assert {
    condition     = strcontains(google_monitoring_alert_policy.http_error_rate[0].conditions[1].condition_threshold[0].filter, "metric.labels.method != \"POST\"")
    error_message = "a one-entry methods list must render like method."
  }
  assert {
    condition     = google_monitoring_alert_policy.http_error_rate[0].conditions[2].display_name == "grype http error rate (POST, non-409)"
    error_message = "a one-entry methods list must render like method: ${google_monitoring_alert_policy.http_error_rate[0].conditions[2].display_name}"
  }
}

run "both_method_forms_rejected" {
  command = plan

  variables {
    http_error_method_status_exclusions = {
      "x" = { method = "GET", methods = ["HEAD"], code = "404" }
    }
  }

  expect_failures = [var.http_error_method_status_exclusions]
}

run "no_method_rejected" {
  command = plan

  variables {
    http_error_method_status_exclusions = {
      "x" = { code = "404" }
    }
  }

  expect_failures = [var.http_error_method_status_exclusions]
}

run "empty_method_rejected" {
  command = plan

  variables {
    http_error_method_status_exclusions = {
      "x" = { methods = ["GET", ""], code = "404" }
    }
  }

  expect_failures = [var.http_error_method_status_exclusions]
}

run "pattern_method_rejected" {
  command = plan

  variables {
    http_error_method_status_exclusions = {
      "x" = { method = "GET|HEAD", code = "404" }
    }
  }

  expect_failures = [var.http_error_method_status_exclusions]
}

run "documentation_defaults_to_summary_line" {
  command = plan

  assert {
    condition     = google_monitoring_alert_policy.http_error_rate[0].documentation[0].content == "$${metric.label.team}: Cloud Run service $${metric_or_resource.labels.job}: HTTP error rate above 0.25"
    error_message = "unexpected default documentation: ${google_monitoring_alert_policy.http_error_rate[0].documentation[0].content}"
  }
}

run "documentation_appended_below_summary_line" {
  command = plan

  variables {
    http_error_documentation = "Runbook text"
  }

  assert {
    condition     = google_monitoring_alert_policy.http_error_rate[0].documentation[0].content == "$${metric.label.team}: Cloud Run service $${metric_or_resource.labels.job}: HTTP error rate above 0.25\n\nRunbook text"
    error_message = "unexpected documentation: ${google_monitoring_alert_policy.http_error_rate[0].documentation[0].content}"
  }
}

run "documentation_over_limit_rejected" {
  command = plan

  variables {
    http_error_documentation = format("%8001s", "x")
  }

  expect_failures = [var.http_error_documentation]
}
