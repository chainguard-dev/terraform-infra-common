/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Package trampoline implements an HTTP server that receives Linear webhook
// events, validates their signatures, and forwards them as CloudEvents to a
// broker ingress.
//
// Use [NewServer] to create a [Server] configured with a CloudEvents client
// and webhook secrets. The server implements [http.Handler] and can be
// registered with any HTTP mux.
//
// # CloudEvent extensions
//
// All forwarded events carry these extensions:
//   - action — Linear action verb: "create" | "update" | "remove"
//   - webhookid — Linear webhook ID
//
// Issue events additionally carry:
//   - issueid — Linear issue UUID (workqueue key for downstream reconcilers)
//   - team — Linear team key (e.g. "ENG")
//
// Comment events additionally carry:
//   - issueid — parent issue UUID
//   - team — derived from the comment URL
//   - authorid — Linear user UUID of the comment author
//   - authorname — display name of the comment author (drift-prone; prefer
//     authorid for filters)
//
// Issue events with action="update" additionally carry per-field boolean
// extensions for each field that changed in the update, sourced from
// Linear's updatedFrom payload field. Subscriptions filter on these to
// drop uninteresting updates (e.g. assignee-only edits) without paying a
// workqueue dispatch + reconcile pass per webhook:
//
//   - updateddescription — issue body changed
//   - updatedstate — workflow state changed (status update)
//   - updatedtitle — title changed
//   - updatedassignee — assignee changed
//   - updatedlabels — label set changed
//   - updatedpriority — priority changed
//   - updatedparent — parent issue changed
//   - updatedcycle — cycle assignment changed
//   - updatedproject — project assignment changed
//
// Fields not enumerated above are not exposed; add a row to
// updatedFieldExtensions in server.go when a downstream consumer needs to
// filter on a new one. Extensions are not set for action="create" or
// action="remove" events — those are scoped to the action= filter alone.
package trampoline
