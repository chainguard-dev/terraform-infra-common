/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Package trampoline implements an HTTP server that receives Cloud Monitoring
// incident notifications from an unwrapped Pub/Sub push subscription and
// forwards each one to a broker ingress as a CloudEvent of type [EventType].
//
// The event data is the notification body exactly as Cloud Monitoring
// published it (https://docs.cloud.google.com/monitoring/support/notification-options#schema-pubsub),
// so consumers see every field, including resource and metric labels.
//
// # CloudEvent attributes
//
//   - id — the Pub/Sub message ID, so a redelivered message keeps its ID
//   - subject — the incident ID
//
// # CloudEvent extensions
//
// Each extension is set only when the notification carries the field:
//   - state — "open" or "closed"
//   - project — the scoping project ID
//   - resourcetype — the monitored resource type (e.g. "cloud_run_revision")
//   - policyname — the alert policy display name
//   - severity — the alert policy severity
package trampoline
