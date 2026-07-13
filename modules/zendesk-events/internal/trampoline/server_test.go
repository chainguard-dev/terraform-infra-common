/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package trampoline

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/jonboulle/clockwork"
)

const (
	testSecret   = "test-webhook-secret"
	testTime     = "2025-01-01T00:00:00Z"
	testTicketID = "35436"
)

type fakeClient struct {
	cloudevents.Client
	events     []cloudevents.Event
	sendResult cloudevents.Result // nil means ACK (success)
}

func (f *fakeClient) Send(_ context.Context, event cloudevents.Event) cloudevents.Result {
	f.events = append(f.events, event)
	return f.sendResult
}

// sign reproduces Zendesk's signature: base64(HMAC-SHA256(secret, timestamp+body)).
func sign(timestamp, body string) string {
	return signWith(testSecret, timestamp, body)
}

// signWith signs (timestamp+body) with an arbitrary secret.
func signWith(secret, timestamp, body string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp))
	mac.Write([]byte(body))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// newTicketPayload builds an event-subscription payload for a ticket event
// (always for testTicketID; callers vary only the event type).
func newTicketPayload(eventType string) string {
	return fmt.Sprintf(
		`{"account_id":12345,"id":%q,"subject":%q,"time":%q,"type":%q,"zendesk_event_version":"2022-11-06","detail":{"id":%q,"status":"open","priority":"normal"},"event":{}}`,
		"evt-"+testTicketID, ticketSubjectPrefix+testTicketID, testTime, eventType, testTicketID,
	)
}

func newRequest(timestamp, body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set(signatureHeader, sign(timestamp, body))
	req.Header.Set(signatureTimestampHeader, timestamp)
	return req
}

func newServer(t *testing.T, secrets ...string) (*Server, *fakeClient) {
	t.Helper()
	clock := clockwork.NewFakeClockAt(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))
	client := &fakeClient{}
	bs := make([][]byte, len(secrets))
	for i, s := range secrets {
		bs[i] = []byte(s)
	}
	s := NewServer(client, bs)
	s.clock = clock
	return s, client
}

func TestServeHTTP_validRequest(t *testing.T) {
	s, client := newServer(t, testSecret)

	body := newTicketPayload("zen:event-type:ticket.created")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, newRequest(testTime, body))

	if w.Code != http.StatusOK {
		t.Fatalf("w.Code = %d, want %d: %s", w.Code, http.StatusOK, w.Body.String())
	}
	if len(client.events) != 1 {
		t.Fatalf("len(client.events) = %d, want 1", len(client.events))
	}

	event := client.events[0]
	if got, want := event.Type(), "dev.chainguard.zendesk.ticket.created"; got != want {
		t.Errorf("type = %s, want %s", got, want)
	}
	if got, want := event.ID(), "evt-35436"; got != want {
		t.Errorf("ID = %s, want %s", got, want)
	}
	if got, want := event.Subject(), "zen:ticket:35436"; got != want {
		t.Errorf("subject = %s, want %s", got, want)
	}
	if got, _ := event.Extensions()["accountid"].(string); got != "12345" {
		t.Errorf("accountid = %q, want 12345", got)
	}
	if got, _ := event.Extensions()["ticketid"].(string); got != "35436" {
		t.Errorf("ticketid = %q, want 35436", got)
	}

	var data eventData
	if err := json.Unmarshal(event.Data(), &data); err != nil {
		t.Fatalf("failed to unmarshal event data: %v", err)
	}
	if data.Headers.EventID != "evt-35436" {
		t.Errorf("headers.event_id = %s, want evt-35436", data.Headers.EventID)
	}
	if data.Headers.EventType != "zen:event-type:ticket.created" {
		t.Errorf("headers.event_type = %s, want zen:event-type:ticket.created", data.Headers.EventType)
	}
	if data.Headers.SignatureTimestamp != testTime {
		t.Errorf("headers.signature_timestamp = %s, want %s", data.Headers.SignatureTimestamp, testTime)
	}
}

func TestServeHTTP_invalidSignature(t *testing.T) {
	s, _ := newServer(t, testSecret)

	body := newTicketPayload("zen:event-type:ticket.created")
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set(signatureHeader, base64.StdEncoding.EncodeToString([]byte("not-the-right-signature")))
	req.Header.Set(signatureTimestampHeader, testTime)

	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("w.Code = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestServeHTTP_missingSignature(t *testing.T) {
	s, _ := newServer(t, testSecret)

	body := newTicketPayload("zen:event-type:ticket.created")
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set(signatureTimestampHeader, testTime)

	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("w.Code = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestServeHTTP_missingTimestamp(t *testing.T) {
	s, _ := newServer(t, testSecret)

	body := newTicketPayload("zen:event-type:ticket.created")
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set(signatureHeader, sign(testTime, body))

	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("w.Code = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestServeHTTP_invalidTimestamp(t *testing.T) {
	s, _ := newServer(t, testSecret)

	// A valid signature over an unparseable timestamp must still be rejected
	// once timestamp parsing fails.
	body := newTicketPayload("zen:event-type:ticket.created")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, newRequest("not-a-timestamp", body))
	if w.Code != http.StatusForbidden {
		t.Fatalf("w.Code = %d, want %d: %s", w.Code, http.StatusForbidden, w.Body.String())
	}
}

func TestServeHTTP_expiredTimestamp(t *testing.T) {
	s, _ := newServer(t, testSecret)

	// 10 minutes before the fake clock, beyond the 5 minute window.
	old := "2024-12-31T23:50:00Z"
	body := newTicketPayload("zen:event-type:ticket.created")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, newRequest(old, body))
	if w.Code != http.StatusForbidden {
		t.Fatalf("w.Code = %d, want %d: %s", w.Code, http.StatusForbidden, w.Body.String())
	}
}

func TestServeHTTP_futureTimestamp(t *testing.T) {
	s, _ := newServer(t, testSecret)

	// 10 minutes after the fake clock, beyond the 5 minute window. A valid
	// signature over a far-future timestamp (clock skew / replay) must still be
	// rejected — exercises the diff > maxTimestampAge half of the window check.
	future := "2025-01-01T00:10:00Z"
	body := newTicketPayload("zen:event-type:ticket.created")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, newRequest(future, body))
	if w.Code != http.StatusForbidden {
		t.Fatalf("w.Code = %d, want %d: %s", w.Code, http.StatusForbidden, w.Body.String())
	}
}

func TestServeHTTP_multipleSecrets(t *testing.T) {
	// The matching secret is second; signing always uses testSecret.
	s, client := newServer(t, "wrong-secret", testSecret)

	body := newTicketPayload("zen:event-type:ticket.created")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, newRequest(testTime, body))

	if w.Code != http.StatusOK {
		t.Fatalf("w.Code = %d, want %d: %s", w.Code, http.StatusOK, w.Body.String())
	}
	if len(client.events) != 1 {
		t.Fatalf("len(client.events) = %d, want 1", len(client.events))
	}
}

func TestServeHTTP_missingEventType(t *testing.T) {
	s, _ := newServer(t, testSecret)

	// Custom trigger/automation payload without the zen:event-type: prefix.
	body := `{"account_id":12345,"id":"evt-1","subject":"zen:ticket:1","type":"custom","time":"2025-01-01T00:00:00Z"}`
	w := httptest.NewRecorder()
	s.ServeHTTP(w, newRequest(testTime, body))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("w.Code = %d, want %d: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestServeHTTP_differentEventTypes(t *testing.T) {
	s, client := newServer(t, testSecret)

	for _, tc := range []struct {
		eventType string
		wantType  string
	}{
		{"zen:event-type:ticket.created", "dev.chainguard.zendesk.ticket.created"},
		{"zen:event-type:ticket.status_changed", "dev.chainguard.zendesk.ticket.status_changed"},
		{"zen:event-type:ticket.comment_added", "dev.chainguard.zendesk.ticket.comment_added"},
		{"zen:event-type:ticket.priority_changed", "dev.chainguard.zendesk.ticket.priority_changed"},
	} {
		t.Run(tc.eventType, func(t *testing.T) {
			client.events = nil
			body := newTicketPayload(tc.eventType)
			w := httptest.NewRecorder()
			s.ServeHTTP(w, newRequest(testTime, body))

			if w.Code != http.StatusOK {
				t.Fatalf("w.Code = %d, want %d: %s", w.Code, http.StatusOK, w.Body.String())
			}
			if len(client.events) != 1 {
				t.Fatalf("len(client.events) = %d, want 1", len(client.events))
			}
			if got := client.events[0].Type(); got != tc.wantType {
				t.Errorf("type = %s, want %s", got, tc.wantType)
			}
			if got, _ := client.events[0].Extensions()["ticketid"].(string); got != "35436" {
				t.Errorf("ticketid = %q, want 35436", got)
			}
		})
	}
}

// TestServeHTTP_redactsBodyAndSubject asserts that the server actually invokes
// redaction on the forwarded event — both redact.Body on the data body and
// redact.String on the CloudEvent subject attribute. A regression that dropped
// either call (or redacted the wrong field) would otherwise pass the rest of
// the suite, since every other test payload is PII-free.
func TestServeHTTP_redactsBodyAndSubject(t *testing.T) {
	s, client := newServer(t, testSecret)

	body := fmt.Sprintf(`{"account_id":12345,"id":"evt-35436",`+
		`"subject":"contact ada@acme.com from 10.1.2.3","time":%q,`+
		`"type":"zen:event-type:ticket.created",`+
		`"detail":{"id":"35436","status":"open","description":"Ada Lovelace cannot pull",`+
		`"tags":["customer:acme.com","cri:regression"]},`+
		`"event":{"comment":{"author_id":"99","body":"My email is ada@acme.com"}}}`, testTime)

	w := httptest.NewRecorder()
	s.ServeHTTP(w, newRequest(testTime, body))
	if w.Code != http.StatusOK {
		t.Fatalf("w.Code = %d, want %d: %s", w.Code, http.StatusOK, w.Body.String())
	}
	if len(client.events) != 1 {
		t.Fatalf("len(client.events) = %d, want 1", len(client.events))
	}
	event := client.events[0]

	// The subject attribute is regex-scrubbed (redact.String).
	if got, want := event.Subject(), "contact <EMAIL> from <IP>"; got != want {
		t.Errorf("subject = %q, want %q", got, want)
	}

	// The data body is redacted (redact.Body): free-text dropped, customer tag
	// masked, technical signal retained, and no raw PII anywhere.
	var data eventData
	if err := json.Unmarshal(event.Data(), &data); err != nil {
		t.Fatalf("failed to unmarshal event data: %v", err)
	}
	bodyStr := string(data.Body)
	for _, leak := range []string{"ada@acme.com", "10.1.2.3", "Ada Lovelace", "customer:acme.com", "cannot pull"} {
		if strings.Contains(bodyStr, leak) {
			t.Errorf("PII %q leaked into forwarded body: %s", leak, bodyStr)
		}
	}
	for _, key := range []string{`"subject"`, `"description"`, `"body"`} {
		if strings.Contains(bodyStr, key) {
			t.Errorf("free-text key %q should have been dropped from body: %s", key, bodyStr)
		}
	}
	// Decode the body to check retained values semantically (the outer event
	// marshal HTML-escapes "<"/">" in the redaction tokens, so substring checks
	// on the raw bytes would miss them).
	var parsed struct {
		Detail struct {
			Status string   `json:"status"`
			Tags   []string `json:"tags"`
		} `json:"detail"`
		Event struct {
			Comment struct {
				AuthorID string `json:"author_id"`
			} `json:"comment"`
		} `json:"event"`
	}
	if err := json.Unmarshal(data.Body, &parsed); err != nil {
		t.Fatalf("failed to unmarshal redacted body: %v", err)
	}
	if parsed.Detail.Status != "open" {
		t.Errorf("detail.status = %q, want open", parsed.Detail.Status)
	}
	if parsed.Event.Comment.AuthorID != "99" {
		t.Errorf("event.comment.author_id = %q, want 99", parsed.Event.Comment.AuthorID)
	}
	if want := []string{"customer:<CUSTOMER>", "cri:regression"}; !slices.Equal(parsed.Detail.Tags, want) {
		t.Errorf("detail.tags = %v, want %v", parsed.Detail.Tags, want)
	}
}

// TestServeHTTP_rejectsNonTicketResource asserts that a validly-signed event for
// a non-ticket resource is rejected with 400 and never forwarded. Only ticket.*
// events are modelled by the recorder schemas and safe under the redaction
// allowlist, so other resources (e.g. agent.*, organization.*) must be dropped
// at the trampoline rather than persisted with un-redacted free-text fields.
func TestServeHTTP_rejectsNonTicketResource(t *testing.T) {
	s, client := newServer(t, testSecret)

	body := fmt.Sprintf(
		`{"account_id":12345,"id":"evt-agent-1","subject":"zen:agent:1","time":%q,"type":"zen:event-type:agent.state_changed"}`,
		testTime,
	)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, newRequest(testTime, body))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("w.Code = %d, want %d: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
	if len(client.events) != 0 {
		t.Fatalf("non-ticket event was forwarded: %d events", len(client.events))
	}
}

// TestServeHTTP_constantSource asserts the CloudEvent source is a stable
// constant rather than the request Host header (which is attacker-controllable
// when the trampoline accepts INGRESS_TRAFFIC_ALL).
func TestServeHTTP_constantSource(t *testing.T) {
	s, client := newServer(t, testSecret)

	body := newTicketPayload("zen:event-type:ticket.created")
	req := newRequest(testTime, body)
	req.Host = "attacker.example.com"

	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("w.Code = %d, want %d: %s", w.Code, http.StatusOK, w.Body.String())
	}
	if got, want := client.events[0].Source(), eventSource; got != want {
		t.Errorf("source = %q, want %q", got, want)
	}
}

// TestServeHTTP_oversizedBody asserts that a body larger than maxBodyBytes is
// rejected before HMAC verification, bounding unauthenticated memory cost.
func TestServeHTTP_oversizedBody(t *testing.T) {
	s, _ := newServer(t, testSecret)

	body := strings.Repeat("A", maxBodyBytes+1024)
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set(signatureHeader, sign(testTime, body))
	req.Header.Set(signatureTimestampHeader, testTime)

	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("w.Code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// TestNewServer_rejectsPlaceholderSecret asserts the pipeline is fail-closed
// during the bootstrap window: when the only configured secret is the public
// "placeholder" value, NewServer discards it, so a request signed with that
// known value is rejected rather than forwarded.
func TestNewServer_rejectsPlaceholderSecret(t *testing.T) {
	client := &fakeClient{}
	s := NewServer(client, [][]byte{[]byte("placeholder")})
	s.clock = clockwork.NewFakeClockAt(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))

	body := newTicketPayload("zen:event-type:ticket.created")
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set(signatureHeader, signWith("placeholder", testTime, body))
	req.Header.Set(signatureTimestampHeader, testTime)

	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("w.Code = %d, want %d: %s", w.Code, http.StatusForbidden, w.Body.String())
	}
	if len(client.events) != 0 {
		t.Fatalf("forged event was forwarded: %d events", len(client.events))
	}
}

// TestNewServer_placeholderAlongsideRealSecret asserts that a real secret still
// works when a placeholder is also present (rotation/bootstrap overlap), while
// the placeholder itself remains unusable for forging.
func TestNewServer_placeholderAlongsideRealSecret(t *testing.T) {
	client := &fakeClient{}
	s := NewServer(client, [][]byte{[]byte("placeholder"), []byte(testSecret)})
	s.clock = clockwork.NewFakeClockAt(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))

	body := newTicketPayload("zen:event-type:ticket.created")

	// Real secret is accepted.
	w := httptest.NewRecorder()
	s.ServeHTTP(w, newRequest(testTime, body))
	if w.Code != http.StatusOK {
		t.Fatalf("real-secret request: w.Code = %d, want %d: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Placeholder-signed request is still rejected.
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set(signatureHeader, signWith("placeholder", testTime, body))
	req.Header.Set(signatureTimestampHeader, testTime)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("placeholder-signed request: w.Code = %d, want %d", w.Code, http.StatusForbidden)
	}
}

// TestServeHTTP_brokerNACK asserts that a NACK or undelivered result from the
// CloudEvents client propagates as HTTP 500, so Zendesk retries the delivery
// rather than silently losing the event.
func TestServeHTTP_brokerNACK(t *testing.T) {
	s, client := newServer(t, testSecret)
	client.sendResult = errors.New("broker unavailable")

	body := newTicketPayload("zen:event-type:ticket.created")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, newRequest(testTime, body))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("w.Code = %d, want %d: %s", w.Code, http.StatusInternalServerError, w.Body.String())
	}
}

func TestTicketIDFromSubject(t *testing.T) {
	tests := []struct {
		subject string
		want    string
	}{
		{"zen:ticket:35436", "35436"},
		{"zen:ticket:1", "1"},
		{"zen:agent:1", ""},
		{"", ""},
		{"35436", ""},
	}
	for _, tc := range tests {
		if got := ticketIDFromSubject(tc.subject); got != tc.want {
			t.Errorf("ticketIDFromSubject(%q) = %q, want %q", tc.subject, got, tc.want)
		}
	}
}

func TestValidateSignature(t *testing.T) {
	body := []byte(`{"test":"data"}`)
	ts := "2025-01-01T00:00:00Z"
	secret := []byte("my-secret")

	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(ts))
	mac.Write(body)
	validSig := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	tests := []struct {
		name      string
		body      []byte
		timestamp string
		signature string
		secrets   [][]byte
		want      bool
	}{
		{"valid signature", body, ts, validSig, [][]byte{secret}, true},
		{"wrong timestamp", body, "2025-01-01T00:00:01Z", validSig, [][]byte{secret}, false},
		{"tampered body", []byte(`{"test":"evil"}`), ts, validSig, [][]byte{secret}, false},
		{"non-base64 signature", body, ts, "not base64!!", [][]byte{secret}, false},
		{"wrong secret", body, ts, validSig, [][]byte{[]byte("wrong-secret")}, false},
		{"second secret matches", body, ts, validSig, [][]byte{[]byte("wrong"), secret}, true},
		{"no secrets", body, ts, validSig, nil, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := validateSignature(tc.body, tc.timestamp, tc.signature, tc.secrets); got != tc.want {
				t.Errorf("validateSignature() = %v, want %v", got, tc.want)
			}
		})
	}
}
