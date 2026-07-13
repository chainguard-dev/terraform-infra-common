/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Package redact scrubs customer-identifying information from a Zendesk webhook
// body before the trampoline forwards it as a CloudEvent — so nothing
// identifying lands in the broker, GCS, or BigQuery. The trampoline forwards
// ticket.* events only, so the allowlists below are tuned to ticket payloads.
//
// # Deny-by-default walk
//
// The body is decoded and walked. A leaf survives only if:
//   - it is a string whose object key is in safeStringKeys (opaque IDs, enum
//     values, timestamps, versions), or
//   - it is a number whose key is in safeNumberKeys (account_id), or
//   - it is a tag with an allowlisted prefix (safeTagPrefixes), or
//   - it is a boolean or null (opaque, e.g. is_public).
//
// Every other leaf — subjects, comment bodies, descriptions, requester
// names/emails, string or numeric custom-field values, free-form tags, and any
// field not enumerated in the recorder schema — is dropped before it can reach
// the broker / GCS / BigQuery. The safe-key allowlists are derived from the
// cloudevent-recorder schemas (schemas/*.schema.json): every STRING column
// except the free-text ones (subject, external_id, comment
// body/html_body/plain_body), plus account_id, the sole INTEGER column.
//
// # Regex pass
//
// After the walk, a regex pass ([String]) scrubs the retained fields and tags
// for identifying tokens that can still occur within technical signal: emails,
// per-customer registry paths, IPv4 and IPv6 addresses, the value of
// "customer:<...>" tags, and a handful of secret formats. Token replacements
// ("<EMAIL>", etc.) contain no JSON metacharacters, so applying them to the
// serialized bytes keeps the document valid.
//
// # Fail-closed
//
// If the regex pass produces invalid JSON, [Body] re-walks with every string
// dropped and re-runs the regex pass over that output; a non-object input is
// dropped to "{}". Every fail-closed path increments zendesk_redact_fail_closed_total
// so the degradation is observable.
//
// Numeric identifiers that are retained (organization_id, requester_id,
// account_id, …) are opaque, carry no name/company, and the organization_id in
// particular is the intended anonymous clustering key (reversible only by
// cross-referencing Zendesk).
package redact
