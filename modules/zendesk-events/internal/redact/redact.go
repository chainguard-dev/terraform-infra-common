/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package redact

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"regexp"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// failClosedTotal counts how often Body fell back to dropping fields rather
// than emitting partially-scrubbed output, labelled by reason. On this
// privacy-critical path the counter lets operators alert on and measure how
// often the scrubber degrades to fail-closed during an incident.
var failClosedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "zendesk_redact_fail_closed_total",
	Help: "Number of times redact.Body failed closed (dropped fields) instead of forwarding scrubbed output.",
}, []string{"reason"})

var (
	emailRE        = regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}`)
	cgrPathRE      = regexp.MustCompile(`((?:packages\.)?cgr\.dev/)([A-Za-z0-9][A-Za-z0-9._-]*)`)
	cgrSubdomainRE = regexp.MustCompile(`\b([A-Za-z0-9][A-Za-z0-9-]*)\.cgr\.dev\b`)
	ipv4RE         = regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)
	// ipv6RE matches full and "::"-compressed IPv6 literals. Every alternative
	// requires either eight colon-separated hextet groups or a "::" run, so
	// colon-bearing non-addresses (RFC3339 timestamps, "12:34:56") don't match.
	// Compiled POSIX (leftmost-longest) so an address like "2001:db8::1" matches
	// in full rather than stopping at the shorter trailing-"::" alternative;
	// POSIX also rejects non-capturing "(?:)", hence the plain groups.
	ipv6RE = regexp.MustCompilePOSIX(`([0-9A-Fa-f]{1,4}:){7}[0-9A-Fa-f]{1,4}|([0-9A-Fa-f]{1,4}:){1,7}:|([0-9A-Fa-f]{1,4}:){1,6}:[0-9A-Fa-f]{1,4}|([0-9A-Fa-f]{1,4}:){1,5}(:[0-9A-Fa-f]{1,4}){1,2}|([0-9A-Fa-f]{1,4}:){1,4}(:[0-9A-Fa-f]{1,4}){1,3}|([0-9A-Fa-f]{1,4}:){1,3}(:[0-9A-Fa-f]{1,4}){1,4}|([0-9A-Fa-f]{1,4}:){1,2}(:[0-9A-Fa-f]{1,4}){1,5}|[0-9A-Fa-f]{1,4}:(:[0-9A-Fa-f]{1,4}){1,6}|:((:[0-9A-Fa-f]{1,4}){1,7}|:)`)
	// customerTagRE matches the value of a Zendesk "customer:<domain>" tag.
	customerTagRE = regexp.MustCompile(`customer:[A-Za-z0-9.\-_]+`)

	secretREs = []*regexp.Regexp{
		regexp.MustCompile(`ghp_[A-Za-z0-9]{36}`),
		regexp.MustCompile(`github_pat_[A-Za-z0-9_]{82}`),
		regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
		regexp.MustCompile(`npm_[A-Za-z0-9]{36}`),
		regexp.MustCompile(`xox[abp]-[A-Za-z0-9-]{10,}`),
		regexp.MustCompile(`sk-[A-Za-z0-9]{20,}`),
		regexp.MustCompile(`eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}`),
		regexp.MustCompile(`(?i)Bearer\s+[A-Za-z0-9._~+/=-]{20,}`),
		regexp.MustCompile(`(?s)-----BEGIN (?:RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----.*?-----END (?:RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----`),
	}

	// nonCustomerRegistries are cgr.dev namespaces that are NOT customers and
	// must be preserved (they are part of the technical signal).
	nonCustomerRegistries = map[string]struct{}{
		"chainguard": {}, "chainguard-private": {}, "wolfi": {}, "chainguard-dev": {},
	}
)

// safeStringKeys enumerates JSON object keys whose string values are known
// technical signal — opaque IDs, enum values, timestamps, and versions — that
// are never customer-authored free text. Under the deny-by-default walk a
// string leaf is kept only if its key appears here; every other string is
// dropped before the body is forwarded.
//
// The set is derived from the recorder schemas (schemas/*.schema.json): every
// STRING column except the free-text ones (subject, external_id, and the
// comment body/html_body/plain_body fields), which carry customer prose.
var safeStringKeys = map[string]struct{}{
	// Top-level routing / versioning.
	"id": {}, "time": {}, "type": {}, "zendesk_event_version": {},
	// detail.* enums and timestamps.
	"status": {}, "priority": {}, "created_at": {}, "updated_at": {},
	// detail.* / event.comment.* identifiers (numeric IDs encoded as strings).
	"brand_id": {}, "form_id": {}, "group_id": {}, "organization_id": {},
	"assignee_id": {}, "requester_id": {}, "submitter_id": {}, "author_id": {},
	// event.previous / event.current carry the status/priority enum values for
	// ticket.status_changed / ticket.priority_changed. These keys are generic
	// and only enum-safe for ticket events (on e.g. organization.name_changed
	// they hold the org name); the trampoline forwards ticket.* events only, so
	// under that contract they only ever hold enums here.
	"previous": {}, "current": {},
}

// safeNumberKeys enumerates JSON object keys whose numeric values are known
// technical signal. account_id is the only INTEGER column in the recorder
// schemas; every other numeric-looking field (organization_id, requester_id, …)
// arrives as a string. Under deny-by-default a number leaf is kept only if its
// key appears here — a number under any other key (e.g. a numeric custom field
// that could carry a phone or account number) is dropped.
var safeNumberKeys = map[string]struct{}{
	"account_id": {},
}

// tagsKey is handled specially in the walk: Zendesk tags are arbitrary
// operator/customer-defined tokens, so under deny-by-default the array is
// filtered to elements whose prefix is in safeTagPrefixes; everything else
// (company names, codenames, internal hostnames) is dropped.
const tagsKey = "tags"

// safeTagPrefixes are the only tag families kept under deny-by-default. "cri:*"
// are root-cause labels (pure technical signal); "customer:*" is retained but
// its value is masked to "customer:<CUSTOMER>" by the regex pass, preserving the
// "this is a customer ticket" signal without the domain. Every other tag is
// free-form and dropped. Widen this set to persist more tag families.
var safeTagPrefixes = []string{"cri:", "customer:"}

// Body returns b with customer-identifying information removed.
//
// It is deny-by-default: the body is decoded and walked, every string leaf
// whose key is not in safeStringKeys is dropped (the tags array is kept), and
// the surviving document is then regex-scrubbed for identifying tokens that can
// still occur within retained technical fields or tags.
//
// If the regex pass produces invalid JSON (e.g. the multi-line private-key
// regex spans two adjacent retained string values and devours the structural
// punctuation between them), Body fails closed: it re-walks with every string
// dropped — guaranteed valid, schema-compatible, and free of any prose — and
// emits a warning so the degradation is observable. An empty or non-object
// input is returned unchanged / dropped.
func Body(b []byte) []byte {
	if len(b) == 0 {
		return b
	}
	walked, ok := walk(b, false)
	if !ok {
		// Not a decodable JSON object: drop the body rather than forward
		// un-scrubbed bytes. (The server only reaches here after a successful
		// unmarshal, so this is a defensive fail-closed.)
		failClosedTotal.WithLabelValues("non_object_body").Inc()
		slog.Warn("redact: body is not a JSON object, dropping it before forward",
			"reason", "non_object_body")
		return []byte("{}")
	}
	s := String(string(walked))
	if json.Valid([]byte(s)) {
		return []byte(s)
	}
	failClosedTotal.WithLabelValues("regex_invalid_json").Inc()
	slog.Warn("redact: regex pass produced invalid JSON, failing closed by dropping all string fields",
		"reason", "regex_invalid_json")
	safe, ok := walk(b, true)
	if !ok {
		return []byte("{}")
	}
	// Run the regex pass over the fallback output too, so any identifying token
	// that occurs inside a JSON *key* (which the walk does not touch) is still
	// scrubbed. The walk dropped every string value, so this normally masks
	// nothing and stays valid; if it somehow does not, drop to "{}".
	safe = []byte(String(string(safe)))
	if !json.Valid(safe) {
		failClosedTotal.WithLabelValues("regex_invalid_json_fallback").Inc()
		return []byte("{}")
	}
	return safe
}

// walk decodes b as a JSON object and rebuilds it under deny-by-default
// redaction. When dropAllStrings is false a string leaf is kept only if its key
// is in safeStringKeys (and the tags array is kept whole); when true every
// string is dropped regardless of key, for the fail-closed fallback. Numbers
// round-trip exactly via json.Number. Reports false if b is not a JSON object.
func walk(b []byte, dropAllStrings bool) ([]byte, bool) {
	dec := json.NewDecoder(bytes.NewReader(b))
	// Decode numbers as json.Number so integer fields (e.g. account_id, which is
	// INTEGER in the recorder schema) round-trip exactly instead of through
	// float64, which would reformat or lose precision on large values.
	dec.UseNumber()
	var v map[string]any
	if err := dec.Decode(&v); err != nil {
		return nil, false
	}
	cleaned := redactObject(v, dropAllStrings)
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	// Disable HTML escaping so any "<"/">" tokens from the regex pass survive
	// byte-for-byte; the encoder otherwise appends a trailing newline we trim.
	enc.SetEscapeHTML(false)
	if err := enc.Encode(cleaned); err != nil {
		return nil, false
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), true
}

// redactObject returns a copy of m keeping only entries that survive
// deny-by-default redaction.
func redactObject(m map[string]any, dropAllStrings bool) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		if k == tagsKey && !dropAllStrings {
			// Deny-by-default for tags too: keep only allowlisted technical
			// prefixes (the regex pass then masks the customer: value); every
			// other free-form tag is dropped.
			if arr, ok := v.([]any); ok {
				out[k] = filterTags(arr)
				continue
			}
		}
		if nv, keep := redactValue(k, v, dropAllStrings); keep {
			out[k] = nv
		}
	}
	return out
}

// redactValue applies deny-by-default redaction to an arbitrary decoded JSON
// value found under key. Objects and arrays are recursed; a string is kept only
// when its key is safe (and dropAllStrings is false); numbers, booleans, and
// null are opaque and kept.
func redactValue(key string, v any, dropAllStrings bool) (any, bool) {
	switch t := v.(type) {
	case map[string]any:
		return redactObject(t, dropAllStrings), true
	case []any:
		return redactArray(key, t, dropAllStrings), true
	case string:
		if dropAllStrings {
			return nil, false
		}
		if _, ok := safeStringKeys[key]; ok {
			return t, true
		}
		return nil, false
	case json.Number:
		// Deny-by-default for numbers too: a numeric custom field could carry a
		// phone or account number, so keep only allowlisted keys.
		if _, ok := safeNumberKeys[key]; ok {
			return v, true
		}
		return nil, false
	default:
		// bool, nil — opaque, carry no customer text.
		return v, true
	}
}

// redactArray redacts each element of a; elements inherit the array's key for
// the safe-key decision.
func redactArray(key string, a []any, dropAllStrings bool) []any {
	out := make([]any, 0, len(a))
	for _, v := range a {
		if nv, keep := redactValue(key, v, dropAllStrings); keep {
			out = append(out, nv)
		}
	}
	return out
}

// filterTags keeps only tag strings whose prefix is in safeTagPrefixes and drops
// everything else (non-string elements included). The retained customer: tag has
// its value masked later by the regex pass.
func filterTags(a []any) []any {
	out := make([]any, 0, len(a))
	for _, v := range a {
		s, ok := v.(string)
		if !ok {
			continue
		}
		for _, p := range safeTagPrefixes {
			if strings.HasPrefix(s, p) {
				out = append(out, s)
				break
			}
		}
	}
	return out
}

// String regex-scrubs s for identifying tokens (emails, per-customer registry
// paths, IPs, "customer:" tag values, and known secret formats). It is the
// second pass over the deny-by-default walk output and is also used directly on
// the CloudEvent subject attribute.
func String(s string) string {
	if s == "" {
		return s
	}
	s = customerTagRE.ReplaceAllString(s, "customer:<CUSTOMER>")
	s = emailRE.ReplaceAllString(s, "<EMAIL>")
	s = cgrPathRE.ReplaceAllStringFunc(s, func(m string) string {
		sub := cgrPathRE.FindStringSubmatch(m)
		if len(sub) != 3 {
			return m
		}
		if _, ok := nonCustomerRegistries[strings.ToLower(sub[2])]; ok {
			return m
		}
		return sub[1] + "<CUSTOMER_REGISTRY>"
	})
	s = cgrSubdomainRE.ReplaceAllStringFunc(s, func(m string) string {
		sub := cgrSubdomainRE.FindStringSubmatch(m)
		if len(sub) != 2 {
			return m
		}
		if _, ok := nonCustomerRegistries[strings.ToLower(sub[1])]; ok {
			return m
		}
		return "<CUSTOMER_REGISTRY>.cgr.dev"
	})
	for _, re := range secretREs {
		s = re.ReplaceAllString(s, "[REDACTED]")
	}
	s = ipv6RE.ReplaceAllString(s, "<IP>")
	s = ipv4RE.ReplaceAllString(s, "<IP>")
	return s
}
