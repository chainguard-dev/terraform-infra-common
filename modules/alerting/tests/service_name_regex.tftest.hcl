# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0
#
# service_name_regex and exclude_service_name_regex are rendered into every
# alert's filter in the syntax of that alert's engine, so a mocked plan shows
# the text Cloud Monitoring receives for each kind.

mock_provider "google" {}

variables {
  project_id = "fixture-project"
  team       = "fixture"
}

run "defaults_leave_filters_and_names_alone" {
  command = plan

  assert {
    condition     = google_monitoring_alert_policy.oom[0].display_name == "OOM Alert fixture"
    error_message = "name must default to the team: ${google_monitoring_alert_policy.oom[0].display_name}"
  }
  assert {
    condition     = !strcontains(google_monitoring_alert_policy.oom[0].conditions[0].condition_matched_log[0].filter, "resource.labels.service_name=~")
    error_message = "no service filter may be rendered when neither regex is set."
  }
  assert {
    condition     = !strcontains(google_monitoring_alert_policy.service_failure_rate_eventing[0].conditions[0].condition_prometheus_query_language[0].query, "job=~")
    error_message = "no job matcher may be rendered when neither regex is set."
  }
}

run "include_regex_scopes_every_alert_kind" {
  command = plan

  variables {
    name               = "carved"
    service_name_regex = "carved-.*|ci-carved"
  }

  assert {
    condition     = google_monitoring_alert_policy.oom[0].display_name == "OOM Alert carved"
    error_message = "name must replace the team in display names: ${google_monitoring_alert_policy.oom[0].display_name}"
  }
  # Logging: anchored, and both the service and the job label, since the
  # log-based alerts cover revisions and jobs alike.
  assert {
    condition     = strcontains(google_monitoring_alert_policy.oom[0].conditions[0].condition_matched_log[0].filter, "(resource.labels.service_name=~\"^(?:carved-.*|ci-carved)$\" OR resource.labels.job_name=~\"^(?:carved-.*|ci-carved)$\")")
    error_message = "the OOM filter must be scoped to the named services and jobs: ${google_monitoring_alert_policy.oom[0].conditions[0].condition_matched_log[0].filter}"
  }
  # The audit-log alerts match the team through protoPayload but still carry
  # the resource labels, so they take the same service filter.
  assert {
    condition     = strcontains(google_monitoring_alert_policy.bad-rollout[0].conditions[0].condition_matched_log[0].filter, "resource.labels.service_name=~\"^(?:carved-.*|ci-carved)$\"")
    error_message = "the rollout filter must be scoped to the named services: ${google_monitoring_alert_policy.bad-rollout[0].conditions[0].condition_matched_log[0].filter}"
  }
  assert {
    condition     = strcontains(google_monitoring_alert_policy.nonzero-exitcode-job[0].conditions[0].condition_matched_log[0].filter, "resource.labels.job_name=~\"^(?:carved-.*|ci-carved)$\"")
    error_message = "the job exit-code filter must be scoped to the named jobs: ${google_monitoring_alert_policy.nonzero-exitcode-job[0].conditions[0].condition_matched_log[0].filter}"
  }
  # Monitoring filter: the Prometheus target's job label is the service name.
  assert {
    condition     = strcontains(google_monitoring_alert_policy.http_error_rate[0].conditions[0].condition_threshold[0].filter, "resource.labels.job=monitoring.regex.full_match(\"carved-.*|ci-carved\")")
    error_message = "the http error filter must be scoped by job: ${google_monitoring_alert_policy.http_error_rate[0].conditions[0].condition_threshold[0].filter}"
  }
  # Dead-letter topics are named after their trigger or workqueue.
  assert {
    condition     = strcontains(google_monitoring_alert_policy.pubsub_dead_letter_queue_messages[0].conditions[0].condition_threshold[0].filter, "metadata.system_labels.\"name\"=monitoring.regex.full_match(\"(?:carved-.*|ci-carved)-dlq-.*\")")
    error_message = "the DLQ filter must be scoped by topic name: ${google_monitoring_alert_policy.pubsub_dead_letter_queue_messages[0].conditions[0].condition_threshold[0].filter}"
  }
  # PromQL: the matcher follows the team matcher inside every selector.
  assert {
    condition     = strcontains(google_monitoring_alert_policy.service_failure_rate_eventing[0].conditions[0].condition_prometheus_query_language[0].query, ", team=\"fixture\", job=~\"carved-.*|ci-carved\"}")
    error_message = "the eventing 5xx query must be scoped by job: ${google_monitoring_alert_policy.service_failure_rate_eventing[0].conditions[0].condition_prometheus_query_language[0].query}"
  }
  assert {
    condition     = strcontains(google_monitoring_alert_policy.grpc_service_failure_rate[0].conditions[0].condition_prometheus_query_language[0].query, ", team=\"fixture\", job=~\"carved-.*|ci-carved\"}")
    error_message = "the grpc failure query must be scoped by job: ${google_monitoring_alert_policy.grpc_service_failure_rate[0].conditions[0].condition_prometheus_query_language[0].query}"
  }
}

run "exclude_regex_negates_every_alert_kind" {
  command = plan

  variables {
    exclude_service_name_regex = "carved-.*|ci-carved"
  }

  assert {
    condition     = google_monitoring_alert_policy.oom[0].display_name == "OOM Alert fixture"
    error_message = "excluding services must not rename the team's policies: ${google_monitoring_alert_policy.oom[0].display_name}"
  }
  assert {
    condition     = strcontains(google_monitoring_alert_policy.oom[0].conditions[0].condition_matched_log[0].filter, "-resource.labels.service_name=~\"^(?:carved-.*|ci-carved)$\"\n-resource.labels.job_name=~\"^(?:carved-.*|ci-carved)$\"")
    error_message = "the OOM filter must leave out the named services and jobs: ${google_monitoring_alert_policy.oom[0].conditions[0].condition_matched_log[0].filter}"
  }
  assert {
    condition     = strcontains(google_monitoring_alert_policy.http_error_rate[0].conditions[0].condition_threshold[0].filter, "resource.labels.job!=monitoring.regex.full_match(\"carved-.*|ci-carved\")")
    error_message = "the http error filter must leave out the named jobs: ${google_monitoring_alert_policy.http_error_rate[0].conditions[0].condition_threshold[0].filter}"
  }
  assert {
    condition     = strcontains(google_monitoring_alert_policy.pubsub_dead_letter_queue_messages[0].conditions[0].condition_threshold[0].filter, "metadata.system_labels.\"name\"!=monitoring.regex.full_match(\"(?:carved-.*|ci-carved)-dlq-.*\")")
    error_message = "the DLQ filter must leave out the named topics: ${google_monitoring_alert_policy.pubsub_dead_letter_queue_messages[0].conditions[0].condition_threshold[0].filter}"
  }
  assert {
    condition     = strcontains(google_monitoring_alert_policy.service_failure_rate_non_eventing[0].conditions[0].condition_prometheus_query_language[0].query, ", team=\"fixture\", job!~\"carved-.*|ci-carved\"}")
    error_message = "the 5xx query must leave out the named jobs: ${google_monitoring_alert_policy.service_failure_rate_non_eventing[0].conditions[0].condition_prometheus_query_language[0].query}"
  }
}

run "invalid_regex_is_refused" {
  command = plan

  variables {
    service_name_regex = "carved-("
  }

  expect_failures = [var.service_name_regex]
}
