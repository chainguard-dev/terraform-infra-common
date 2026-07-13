/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Package trampoline implements an HTTP server that receives Zendesk webhook
// events (from Zendesk event subscriptions), validates their signatures, and
// forwards them as CloudEvents to a broker ingress.
//
// Use [NewServer] to create a [Server] configured with a CloudEvents client
// and webhook secrets. The server implements [http.Handler] and can be
// registered with any HTTP mux.
//
// # Signature verification
//
// Zendesk signs the concatenation of the X-Zendesk-Webhook-Signature-Timestamp
// header and the raw request body with HMAC-SHA256, and sends the result
// base64-encoded in the X-Zendesk-Webhook-Signature header. Both headers are
// required; requests missing either, with an unparseable timestamp, with a
// timestamp outside the replay window, or with a signature that matches none of
// the configured secrets are rejected with 403.
//
// # CloudEvent type
//
// The CloudEvent type is derived from the payload `type` field by stripping the
// "zen:event-type:" prefix and prepending "dev.chainguard.zendesk.". For
// example "zen:event-type:ticket.created" becomes
// "dev.chainguard.zendesk.ticket.created". Payloads without the
// "zen:event-type:" prefix (e.g. custom trigger/automation webhooks) are
// rejected with 400.
//
// Only ticket.* events are forwarded; any other resource (e.g.
// organization.name_changed) is rejected with 400. The recorder ships schemas
// for ticket.* exclusively, and the redaction allowlist is only safe for ticket
// events — see the redact package.
//
// # CloudEvent extensions
//
// All forwarded events carry:
//   - accountid — Zendesk account ID (omitted when zero)
//   - ticketid — numeric ticket ID extracted from the subject
//     ("zen:ticket:<id>"); the immutable workqueue key for downstream
//     reconcilers.
package trampoline
