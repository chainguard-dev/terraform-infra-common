/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package redact

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestBody_redactsAndStaysValidJSON(t *testing.T) {
	in := []byte(`{"account_id":12345,"id":"evt-1","subject":"can't pull, from Ada Lovelace",` +
		`"type":"zen:event-type:ticket.created",` +
		`"detail":{"id":"42","organization_id":"7","requester_id":"99","status":"open","priority":"normal",` +
		`"subject":"can't pull cgr.dev/acme.com/nginx","description":"see acme.cgr.dev and 10.1.2.3","external_id":"ACME-RT-99",` +
		`"tags":["customer:acme.com","cri:regression"]},` +
		`"event":{"comment":{"author_id":"99","body":"Hi, this is Ada at ada@acme.com"}}}`)
	out := Body(in)

	// Still valid JSON.
	var v any
	if err := json.Unmarshal(out, &v); err != nil {
		t.Fatalf("redacted body is not valid JSON: %v\n%s", err, out)
	}

	s := string(out)
	// Free-text and PII (in dropped fields) must not survive anywhere.
	for _, leak := range []string{
		"Ada Lovelace", "ada@acme.com", "cgr.dev/acme.com", "acme.cgr.dev", "10.1.2.3",
		"ACME-RT-99", "customer:acme.com", "can't pull",
	} {
		if strings.Contains(s, leak) {
			t.Errorf("leak %q remained: %s", leak, s)
		}
	}
	// Retained technical signal: numeric ids (clustering keys), enums, and the
	// customer: tag masked while cri: survives.
	for _, keep := range []string{
		`"account_id":12345`, `"id":"evt-1"`, `"organization_id":"7"`, `"requester_id":"99"`,
		`"status":"open"`, `"priority":"normal"`, `"author_id":"99"`,
		"cri:regression", "customer:<CUSTOMER>",
	} {
		if !strings.Contains(s, keep) {
			t.Errorf("expected %q to be preserved: %s", keep, s)
		}
	}
	// Free-text keys are dropped entirely (deny-by-default), not just blanked.
	for _, key := range []string{`"subject"`, `"description"`, `"external_id"`, `"body"`} {
		if strings.Contains(s, key) {
			t.Errorf("free-text key %q should have been dropped: %s", key, s)
		}
	}
}

// TestBody_dropsFreeFormTags asserts the tags array is deny-by-default: only
// allowlisted prefixes (cri:, customer:) survive; arbitrary operator/customer
// tags (company names, codenames, internal hostnames) are dropped even though
// none of the regexes match them.
func TestBody_dropsFreeFormTags(t *testing.T) {
	in := []byte(`{"id":"evt-1","type":"zen:event-type:ticket.created","detail":{"id":"42",` +
		`"tags":["cri:regression","customer:acme.com","acme-corporation","project-thunderbird","bastion-01"]}}`)
	out := Body(in)

	var v any
	if err := json.Unmarshal(out, &v); err != nil {
		t.Fatalf("redacted body is not valid JSON: %v\n%s", err, out)
	}
	s := string(out)
	for _, keep := range []string{"cri:regression", "customer:<CUSTOMER>"} {
		if !strings.Contains(s, keep) {
			t.Errorf("expected allowlisted tag %q to survive: %s", keep, s)
		}
	}
	for _, drop := range []string{"acme-corporation", "project-thunderbird", "bastion-01", "customer:acme.com"} {
		if strings.Contains(s, drop) {
			t.Errorf("free-form/unmasked tag %q should have been dropped: %s", drop, s)
		}
	}
}

// TestBody_dropsUnknownStringFields asserts the deny-by-default posture: a
// string field whose key is not in safeStringKeys is dropped wholesale,
// regardless of whether the regex pass would have matched its contents.
func TestBody_dropsUnknownStringFields(t *testing.T) {
	in := []byte(`{` +
		`"subject":"Jane Doe at Acme says +1 (555) 010-1234 cannot pull",` +
		`"detail":{"subject":"Jane Doe — internal hostname db-prod-01.acme.internal","external_id":"ACME-RT-99",` +
		`"description":"long ticket body naming Jane Doe","status":"open","requester_id":"99"},` +
		`"event":{"comment":{"author_id":"7","body":"Hi, this is Jane Doe","html_body":"<p>Hi, this is Jane Doe</p>","plain_body":"Hi, this is Jane Doe"}}` +
		`}`)
	out := Body(in)

	var v any
	if err := json.Unmarshal(out, &v); err != nil {
		t.Fatalf("redacted body is not valid JSON: %v\n%s", err, out)
	}
	s := string(out)

	// Every free-text key must be absent.
	for _, key := range []string{
		`"subject"`, `"external_id"`, `"description"`, `"body"`, `"html_body"`, `"plain_body"`,
	} {
		if strings.Contains(s, key) {
			t.Errorf("free-text key %q should have been dropped: %s", key, s)
		}
	}
	// And none of the prose survives.
	for _, leak := range []string{"Jane Doe", "Acme", "db-prod-01", "ACME-RT-99", "555"} {
		if strings.Contains(s, leak) {
			t.Errorf("leak %q remained: %s", leak, s)
		}
	}
	// Safe technical fields are retained.
	for _, keep := range []string{`"status":"open"`, `"requester_id":"99"`, `"author_id":"7"`} {
		if !strings.Contains(s, keep) {
			t.Errorf("expected %q to be preserved: %s", keep, s)
		}
	}
}

func TestBody_preservesChainguardRegistries(t *testing.T) {
	// The cgr.dev registry regexes run on a retained field: Chainguard's own
	// namespaces are preserved as technical signal while per-customer registries
	// are masked. (Registry refs in free-text or free-form tags are dropped
	// outright by the deny-by-default walk; this exercises the regex on a
	// retained safeStringKeys field.)
	in := []byte(`{"detail":{"status":"` +
		`cgr.dev/chainguard-private/nginx cgr.dev/chainguard/nginx chainguard.cgr.dev ` +
		`cgr.dev/acme.com/nginx acme.cgr.dev"}}`)
	s := string(Body(in))
	for _, keep := range []string{"cgr.dev/chainguard-private/nginx", "cgr.dev/chainguard/nginx", "chainguard.cgr.dev"} {
		if !strings.Contains(s, keep) {
			t.Errorf("non-customer registry %q wrongly redacted: %s", keep, s)
		}
	}
	for _, leak := range []string{"cgr.dev/acme.com", "acme.cgr.dev"} {
		if strings.Contains(s, leak) {
			t.Errorf("customer registry %q not redacted: %s", leak, s)
		}
	}
	if !strings.Contains(s, "<CUSTOMER_REGISTRY>") {
		t.Errorf("expected <CUSTOMER_REGISTRY> token in: %s", s)
	}
}

func TestBody_empty(t *testing.T) {
	if got := Body(nil); got != nil {
		t.Errorf("Body(nil) = %v, want nil", got)
	}
}

// TestBody_redactsSecretsInRetainedFields asserts that the secretREs patterns
// fire for credential formats that survive the deny-by-default walk (i.e. in a
// retained safeStringKeys field). A typo in any secretRE would otherwise
// silently leak credentials whenever a secret lands in a kept field.
func TestBody_redactsSecretsInRetainedFields(t *testing.T) {
	ghpToken := "ghp_" + strings.Repeat("A", 36)
	akiaKey := "AKIA" + strings.Repeat("B", 16)
	bearerToken := "Bearer " + strings.Repeat("C", 25)

	// status/priority/type are retained (safeStringKeys); a secret pasted into one
	// must still be scrubbed by the regex pass.
	in := []byte(`{"detail":{"status":"` + ghpToken + `","priority":"` + akiaKey + `","type":"` + bearerToken + `"}}`)
	out := Body(in)
	if !json.Valid(out) {
		t.Fatalf("redacted body is not valid JSON: %s", out)
	}
	s := string(out)
	for _, leak := range []string{ghpToken, akiaKey, bearerToken} {
		if strings.Contains(s, leak) {
			t.Errorf("secret %q was not redacted: %s", leak, s)
		}
	}
	if !strings.Contains(s, "[REDACTED]") {
		t.Errorf("expected [REDACTED] placeholder in: %s", s)
	}
}

// TestBody_dropsSecretsInUnknownFields confirms the deny-by-default belt: a
// secret pasted into a non-allowlisted field is dropped outright (it never even
// reaches the regex pass), so it cannot leak even if a secretRE regressed.
func TestBody_dropsSecretsInUnknownFields(t *testing.T) {
	ghpToken := "ghp_" + strings.Repeat("A", 36)
	in := []byte(`{"detail":{"api_token":"` + ghpToken + `","status":"open"}}`)
	out := Body(in)
	if !json.Valid(out) {
		t.Fatalf("redacted body is not valid JSON: %s", out)
	}
	s := string(out)
	if strings.Contains(s, ghpToken) {
		t.Errorf("secret in unknown field was not dropped: %s", s)
	}
	if strings.Contains(s, `"api_token"`) {
		t.Errorf("unknown field key should have been dropped: %s", s)
	}
	if !strings.Contains(s, `"status":"open"`) {
		t.Errorf("expected safe field to be preserved: %s", s)
	}
}

// TestBody_preservesLargeIntegers asserts that integer fields round-trip
// exactly through the walk's decode/re-encode. The body is unmarshalled into
// map[string]any and re-serialized; without json.Number a large account_id
// would decode as float64 and re-emit in exponent form or lose precision,
// silently diverging from what Zendesk sent and from the INTEGER recorder
// schema.
func TestBody_preservesLargeIntegers(t *testing.T) {
	// A value beyond 2^53, where float64 loses integer precision.
	in := []byte(`{"account_id":9007199254740993,"detail":{"status":"open"}}`)
	out := Body(in)
	if !json.Valid(out) {
		t.Fatalf("redacted body is not valid JSON: %s", out)
	}
	if s := string(out); !strings.Contains(s, `"account_id":9007199254740993`) {
		t.Errorf("account_id did not round-trip exactly: %s", s)
	}
}

// TestBody_dropsNumbersUnderUnknownKeys asserts deny-by-default extends to
// numbers: a numeric value under a non-allowlisted key (e.g. a custom field that
// could hold a phone number) is dropped, while the allowlisted account_id and
// safe string fields are retained.
func TestBody_dropsNumbersUnderUnknownKeys(t *testing.T) {
	in := []byte(`{"account_id":12345,"detail":{"custom_phone":15550101234,"status":"open"}}`)
	out := Body(in)
	if !json.Valid(out) {
		t.Fatalf("redacted body is not valid JSON: %s", out)
	}
	s := string(out)
	if strings.Contains(s, "15550101234") || strings.Contains(s, "custom_phone") {
		t.Errorf("numeric field under unknown key should have been dropped: %s", s)
	}
	if !strings.Contains(s, `"account_id":12345`) {
		t.Errorf("account_id should be retained: %s", s)
	}
	if !strings.Contains(s, `"status":"open"`) {
		t.Errorf("safe string field should be retained: %s", s)
	}
}

// TestBody_scrubsIPv6 asserts the regex pass masks IPv6 literals (both full and
// "::"-compressed) in a retained field, and that a colon-bearing timestamp is
// left intact (the pattern requires eight hextets or a "::" run).
func TestBody_scrubsIPv6(t *testing.T) {
	in := []byte(`{"detail":{"status":"seen from 2001:db8::1 and fe80::1ff:fe23:4567:890a","created_at":"2025-01-01T12:34:56Z"}}`)
	out := Body(in)
	if !json.Valid(out) {
		t.Fatalf("redacted body is not valid JSON: %s", out)
	}
	s := string(out)
	for _, leak := range []string{"2001:db8::1", "fe80::1ff:fe23:4567:890a"} {
		if strings.Contains(s, leak) {
			t.Errorf("IPv6 %q was not scrubbed: %s", leak, s)
		}
	}
	if !strings.Contains(s, "<IP>") {
		t.Errorf("expected <IP> token in: %s", s)
	}
	if !strings.Contains(s, "2025-01-01T12:34:56Z") {
		t.Errorf("timestamp should not be mangled by the IPv6 regex: %s", s)
	}
}

// fallbackBody is a payload whose regex pass produces invalid JSON: a private
// key spans a retained field (detail.status) and another retained field across
// a nested-object boundary (event.comment.author_id), so the (?s) BEGIN..END
// regex devours an object's opening brace and unbalances the document. It
// exercises the fail-closed regex fallback.
const fallbackBody = `{"detail":{"status":"-----BEGIN PRIVATE KEY-----"},"event":{"comment":{"author_id":"-----END PRIVATE KEY-----"}}}`

// TestBody_failClosedEmitsMetric asserts that the fail-closed regex fallback
// increments the observability counter so the leak/degradation rate is visible
// during an incident.
func TestBody_failClosedEmitsMetric(t *testing.T) {
	c := failClosedTotal.WithLabelValues("regex_invalid_json")
	before := testutil.ToFloat64(c)

	if out := Body([]byte(fallbackBody)); !json.Valid(out) {
		t.Fatalf("Body returned invalid JSON: %s", out)
	}

	if got := testutil.ToFloat64(c) - before; got != 1 {
		t.Errorf("fail_closed_total{reason=regex_invalid_json} delta = %v, want 1", got)
	}
}

// TestBody_nonObjectFailsClosedEmitsMetric covers the other fail-closed branch:
// a body that is valid JSON but not an object (e.g. a top-level array) cannot be
// walked, so Body drops it to "{}" rather than forwarding un-scrubbed bytes, and
// increments the non_object_body counter so the leak-avoided path is observable.
func TestBody_nonObjectFailsClosedEmitsMetric(t *testing.T) {
	c := failClosedTotal.WithLabelValues("non_object_body")
	before := testutil.ToFloat64(c)

	if out := Body([]byte(`[1,2,3]`)); string(out) != "{}" {
		t.Errorf("Body(non-object) = %q, want {}", out)
	}
	if got := testutil.ToFloat64(c) - before; got != 1 {
		t.Errorf("fail_closed_total{reason=non_object_body} delta = %v, want 1", got)
	}
}

// TestBody_regexFallbackFailsClosed asserts that when the regex pass produces
// invalid JSON, Body actually takes the fail-closed fallback — re-walking with
// every string dropped — rather than emitting a malformed document or the
// pre-regex walk output. It verifies the branch was taken (the retained safe
// fields detail.status / event.comment.author_id, which the normal walk would
// keep, are gone) and that no fragment of the private key survives.
func TestBody_regexFallbackFailsClosed(t *testing.T) {
	out := Body([]byte(fallbackBody))
	if !json.Valid(out) {
		t.Fatalf("Body returned invalid JSON: %s", out)
	}
	s := string(out)

	// No part of the private key — proving the regex output was not returned.
	for _, leak := range []string{"BEGIN PRIVATE KEY", "END PRIVATE KEY", "-----"} {
		if strings.Contains(s, leak) {
			t.Errorf("private-key marker %q survived the fallback: %s", leak, s)
		}
	}
	// The fallback drops ALL strings, so even the otherwise-retained safe values
	// are gone — distinguishing the fail-closed walk from the normal one.
	var parsed struct {
		Detail struct {
			Status string `json:"status"`
		} `json:"detail"`
		Event struct {
			Comment struct {
				AuthorID string `json:"author_id"`
			} `json:"comment"`
		} `json:"event"`
	}
	if err := json.Unmarshal(out, &parsed); err != nil {
		t.Fatalf("fallback output is not valid JSON: %v", err)
	}
	if parsed.Detail.Status != "" {
		t.Errorf("fail-closed fallback should have dropped detail.status, got %q", parsed.Detail.Status)
	}
	if parsed.Event.Comment.AuthorID != "" {
		t.Errorf("fail-closed fallback should have dropped event.comment.author_id, got %q", parsed.Event.Comment.AuthorID)
	}
}
