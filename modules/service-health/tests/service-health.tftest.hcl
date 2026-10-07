# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0
#
# Run in CI by .github/workflows/tf-module-tests.yaml.

mock_provider "google" {}

variables {
  project_id                  = "fixture-project"
  notification_channels_slack = ["projects/fixture-project/notificationChannels/slack-1"]
}

run "notifies_every_channel_type" {
  command = plan
  variables {
    notification_channels_slack = [
      "projects/fixture-project/notificationChannels/slack-1",
      "projects/fixture-project/notificationChannels/slack-2",
    ]
    notification_channels_email  = ["projects/fixture-project/notificationChannels/email-1"]
    notification_channels_pubsub = ["projects/fixture-project/notificationChannels/pubsub-1"]
    notification_channels        = ["projects/fixture-project/notificationChannels/slack-1"]
  }
  assert {
    condition = toset(google_monitoring_alert_policy.incident.notification_channels) == toset([
      "projects/fixture-project/notificationChannels/slack-1",
      "projects/fixture-project/notificationChannels/slack-2",
      "projects/fixture-project/notificationChannels/email-1",
      "projects/fixture-project/notificationChannels/pubsub-1",
    ])
    error_message = "the policy must notify every Slack, email, Pub/Sub, and generic channel"
  }
  assert {
    condition     = length(google_monitoring_alert_policy.incident.notification_channels) == 4
    error_message = "a channel passed in more than one input must only be notified once"
  }
}

run "requires_a_channel" {
  command = plan
  variables {
    notification_channels_slack = []
  }
  expect_failures = [google_monitoring_alert_policy.incident]
}

run "rejects_malformed_channel" {
  command = plan
  variables {
    notification_channels_email = ["email-1"]
  }
  expect_failures = [var.notification_channels_email]
}

run "default_filter_matches_all_incidents" {
  command = plan
  assert {
    condition     = !strcontains(output.filter, "impactedProducts") && !strcontains(output.filter, "impactedLocations")
    error_message = "with no products or locations, the filter must not narrow by either"
  }
  assert {
    condition     = strcontains(output.filter, "jsonPayload.category=\"INCIDENT\"")
    error_message = "the filter must only match incidents"
  }
}

run "filters_by_product_and_location" {
  command = plan
  variables {
    products  = ["Cloud Run", "Google Kubernetes Engine"]
    locations = ["us-central1"]
  }
  assert {
    condition     = strcontains(output.filter, "AND (jsonPayload.impactedProducts:\"Cloud Run\" OR jsonPayload.impactedProducts:\"Google Kubernetes Engine\")")
    error_message = "products must be OR'd together and AND'd with the base filter"
  }
  assert {
    condition     = strcontains(output.filter, "AND (jsonPayload.impactedLocations:\"us-central1\")")
    error_message = "locations must be AND'd with the base filter"
  }
}

run "name_prefixes_subject" {
  command = plan
  variables {
    name = "enforce.dev"
  }
  assert {
    condition     = startswith(google_monitoring_alert_policy.incident.documentation[0].subject, "[enforce.dev] GCP incident")
    error_message = "name must prefix the notification subject"
  }
}

run "api_enablement_is_optional" {
  command = plan
  variables {
    enable_api = false
  }
  assert {
    condition     = length(google_project_service.servicehealth) == 0
    error_message = "enable_api = false must not manage the Service Health API"
  }
}
