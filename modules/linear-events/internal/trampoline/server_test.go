/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package trampoline

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/jonboulle/clockwork"
)

const testSecret = "test-webhook-secret"

type fakeClient struct {
	cloudevents.Client
	events []cloudevents.Event
}

func (f *fakeClient) Send(_ context.Context, event cloudevents.Event) cloudevents.Result {
	f.events = append(f.events, event)
	return nil
}

func sign(body string) string {
	mac := hmac.New(sha256.New, []byte(testSecret))
	mac.Write([]byte(body))
	return hex.EncodeToString(mac.Sum(nil))
}

func newTestPayload(action, entityType string, ts int64) string {
	return fmt.Sprintf(`{"action":%q,"type":%q,"organizationId":"org-123","webhookId":"wh-456","webhookTimestamp":%d,"createdAt":"2025-01-01T00:00:00.000Z","url":"https://linear.app/team/issue/ENG-1","data":{"id":"issue-789","title":"Test issue","identifier":"ENG-1","number":1,"priority":2,"state":{"id":"state-1","name":"In Progress","type":"started"},"team":{"id":"team-1","key":"ENG","name":"Engineering"}}}`, action, entityType, ts)
}

func newCommentPayload(action string, ts int64) string {
	// Mirrors the Linear webhook shape (see schemas/comment.schema.json):
	// top-level `actor` { id, name, type } is the comment author; data.userId
	// duplicates the actor id but carries no name.
	return fmt.Sprintf(`{"action":%q,"type":"Comment","organizationId":"org-123","webhookId":"wh-456","webhookTimestamp":%d,"url":"https://linear.app/chainguard/issue/ENG-1/test-issue#comment-001","actor":{"id":"user-1","name":"Auto Bot","type":"user"},"data":{"id":"comment-001","body":"A comment","issueId":"issue-789","userId":"user-1"}}`, action, ts)
}

func TestServeHTTP_validRequest(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := clockwork.NewFakeClockAt(now)
	client := &fakeClient{}

	s := NewServer(client, [][]byte{[]byte(testSecret)})
	s.clock = clock

	body := newTestPayload("create", "Issue", now.UnixMilli())
	sig := sign(body)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Linear-Signature", sig)
	req.Header.Set("Linear-Event", "Issue")
	req.Header.Set("Linear-Delivery", "delivery-123")

	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("w.Code = %d, want %d: %s", w.Code, http.StatusOK, w.Body.String())
	}

	if len(client.events) != 1 {
		t.Fatalf("len(client.events) = %d, want 1", len(client.events))
	}

	event := client.events[0]
	if got, want := event.Type(), "dev.chainguard.linear.issue"; got != want {
		t.Errorf("type = %s, want %s", got, want)
	}
	if got, want := event.ID(), "delivery-123"; got != want {
		t.Errorf("ID = %s, want %s", got, want)
	}
	if got, want := event.Subject(), "org-123"; got != want {
		t.Errorf("subject = %s, want %s", got, want)
	}

	action, _ := event.Extensions()["action"].(string)
	if action != "create" {
		t.Errorf("action = %s, want create", action)
	}
	webhookID, _ := event.Extensions()["webhookid"].(string)
	if webhookID != "wh-456" {
		t.Errorf("webhookid = %s, want wh-456", webhookID)
	}

	// Verify issue-specific extensions.
	issueURL, _ := event.Extensions()["issueid"].(string)
	if issueURL != "issue-789" {
		t.Errorf("issueid = %s, want issue-789", issueURL)
	}
	team, _ := event.Extensions()["team"].(string)
	if team != "ENG" {
		t.Errorf("team = %s, want ENG", team)
	}

	// Verify event data envelope.
	var data eventData
	if err := json.Unmarshal(event.Data(), &data); err != nil {
		t.Fatalf("failed to unmarshal event data: %v", err)
	}
	if data.Headers.DeliveryID != "delivery-123" {
		t.Errorf("headers.delivery_id = %s, want delivery-123", data.Headers.DeliveryID)
	}
	if data.Headers.Event != "Issue" {
		t.Errorf("headers.event = %s, want Issue", data.Headers.Event)
	}
}

func TestServeHTTP_invalidSignature(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := clockwork.NewFakeClockAt(now)
	client := &fakeClient{}

	s := NewServer(client, [][]byte{[]byte(testSecret)})
	s.clock = clock

	body := newTestPayload("create", "Issue", now.UnixMilli())

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Linear-Signature", "invalid-signature")
	req.Header.Set("Linear-Event", "Issue")
	req.Header.Set("Linear-Delivery", "delivery-123")

	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("w.Code = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestServeHTTP_missingSignature(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := clockwork.NewFakeClockAt(now)
	client := &fakeClient{}

	s := NewServer(client, [][]byte{[]byte(testSecret)})
	s.clock = clock

	body := newTestPayload("create", "Issue", now.UnixMilli())

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Linear-Event", "Issue")
	req.Header.Set("Linear-Delivery", "delivery-123")

	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("w.Code = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestServeHTTP_missingEventHeader(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := clockwork.NewFakeClockAt(now)
	client := &fakeClient{}

	s := NewServer(client, [][]byte{[]byte(testSecret)})
	s.clock = clock

	body := newTestPayload("create", "Issue", now.UnixMilli())
	sig := sign(body)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Linear-Signature", sig)
	req.Header.Set("Linear-Delivery", "delivery-123")

	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("w.Code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestServeHTTP_expiredTimestamp(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := clockwork.NewFakeClockAt(now)
	client := &fakeClient{}

	s := NewServer(client, [][]byte{[]byte(testSecret)})
	s.clock = clock

	// Timestamp 10 minutes ago, beyond the 5 minute window.
	oldTS := now.Add(-10 * time.Minute).UnixMilli()
	body := newTestPayload("update", "Issue", oldTS)
	sig := sign(body)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Linear-Signature", sig)
	req.Header.Set("Linear-Event", "Issue")
	req.Header.Set("Linear-Delivery", "delivery-123")

	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("w.Code = %d, want %d: %s", w.Code, http.StatusForbidden, w.Body.String())
	}
}

func TestServeHTTP_multipleSecrets(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := clockwork.NewFakeClockAt(now)
	client := &fakeClient{}

	s := NewServer(client, [][]byte{[]byte("wrong-secret"), []byte(testSecret)})
	s.clock = clock

	body := newTestPayload("update", "Issue", now.UnixMilli())
	sig := sign(body)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Linear-Signature", sig)
	req.Header.Set("Linear-Event", "Issue")
	req.Header.Set("Linear-Delivery", "delivery-456")

	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("w.Code = %d, want %d: %s", w.Code, http.StatusOK, w.Body.String())
	}
	if len(client.events) != 1 {
		t.Fatalf("len(client.events) = %d, want 1", len(client.events))
	}
}

func TestServeHTTP_differentEventTypes(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := clockwork.NewFakeClockAt(now)
	client := &fakeClient{}

	s := NewServer(client, [][]byte{[]byte(testSecret)})
	s.clock = clock

	for _, tc := range []struct {
		eventHeader string
		wantType    string
	}{{
		"Issue", "dev.chainguard.linear.issue",
	}, {
		"Comment", "dev.chainguard.linear.comment",
	}, {
		"Project", "dev.chainguard.linear.project",
	}, {
		"Cycle", "dev.chainguard.linear.cycle",
	}, {
		"IssueLabel", "dev.chainguard.linear.issuelabel",
	}} {
		t.Run(tc.eventHeader, func(t *testing.T) {
			client.events = nil

			body := newTestPayload("create", tc.eventHeader, now.UnixMilli())
			sig := sign(body)

			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
			req.Header.Set("Linear-Signature", sig)
			req.Header.Set("Linear-Event", tc.eventHeader)
			req.Header.Set("Linear-Delivery", "delivery-"+tc.eventHeader)

			w := httptest.NewRecorder()
			s.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("w.Code = %d, want %d: %s", w.Code, http.StatusOK, w.Body.String())
			}
			if len(client.events) != 1 {
				t.Fatalf("len(client.events) = %d, want 1", len(client.events))
			}
			if got := client.events[0].Type(); got != tc.wantType {
				t.Errorf("type = %s, want %s", got, tc.wantType)
			}
		})
	}
}

// TestServeHTTP_issueUpdatedFieldExtensions verifies that an issue.update
// payload's updatedFrom keys map to the per-field "updated<field>=true"
// CloudEvent extensions downstream subscriptions filter on. A bot that only
// cares about description/state updates declares filter maps for those
// extensions and lets assignee/label/priority-only updates fall on the
// floor at the broker, instead of paying a workqueue dispatch and reconcile
// pass per uninteresting webhook.
func TestServeHTTP_issueUpdatedFieldExtensions(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := clockwork.NewFakeClockAt(now)

	tests := []struct {
		name           string
		updatedFrom    string // raw JSON object body for updatedFrom
		wantExtensions map[string]bool
		wantAbsent     []string // extensions that must NOT be present
	}{
		{
			name:        "description-only update",
			updatedFrom: `{"description":"old body"}`,
			wantExtensions: map[string]bool{
				"updateddescription": true,
			},
			wantAbsent: []string{"updatedstate", "updatedassignee", "updatedlabels"},
		},
		{
			name:        "state-only update (status change)",
			updatedFrom: `{"stateId":"old-state-uuid"}`,
			wantExtensions: map[string]bool{
				"updatedstate": true,
			},
			wantAbsent: []string{"updateddescription", "updatedassignee"},
		},
		{
			name:        "assignee-only update (must NOT trigger description/state)",
			updatedFrom: `{"assigneeId":null}`,
			wantExtensions: map[string]bool{
				"updatedassignee": true,
			},
			wantAbsent: []string{"updateddescription", "updatedstate", "updatedtitle"},
		},
		{
			name:        "multi-field update (description + state)",
			updatedFrom: `{"description":"old","stateId":"old-state","assigneeId":null}`,
			wantExtensions: map[string]bool{
				"updateddescription": true,
				"updatedstate":       true,
				"updatedassignee":    true,
			},
		},
		{
			name:        "unknown field (parentName) does not produce an extension",
			updatedFrom: `{"parentName":"old name"}`,
			wantAbsent:  []string{"updatedparentname", "updateddescription"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := &fakeClient{}
			s := NewServer(client, [][]byte{[]byte(testSecret)})
			s.clock = clock

			body := fmt.Sprintf(
				`{"action":"update","type":"Issue","organizationId":"org-123","webhookId":"wh-456","webhookTimestamp":%d,"createdAt":"2025-01-01T00:00:00.000Z","url":"https://linear.app/team/issue/ENG-1","data":{"id":"issue-789","title":"Test issue","identifier":"ENG-1","number":1,"team":{"id":"team-1","key":"ENG","name":"Engineering"}},"updatedFrom":%s}`,
				now.UnixMilli(),
				tc.updatedFrom,
			)
			sig := sign(body)

			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
			req.Header.Set("Linear-Signature", sig)
			req.Header.Set("Linear-Event", "Issue")
			req.Header.Set("Linear-Delivery", "delivery-update")

			w := httptest.NewRecorder()
			s.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("w.Code = %d, want %d: %s", w.Code, http.StatusOK, w.Body.String())
			}
			if len(client.events) != 1 {
				t.Fatalf("len(client.events) = %d, want 1", len(client.events))
			}

			ext := client.events[0].Extensions()
			for k, want := range tc.wantExtensions {
				got, _ := ext[k].(bool)
				if got != want {
					t.Errorf("extension %q: got = %v, want = %v", k, got, want)
				}
			}
			for _, k := range tc.wantAbsent {
				if _, present := ext[k]; present {
					t.Errorf("extension %q: present = true, want = false (assignee-only updates must not trigger description/state filters)", k)
				}
			}
		})
	}
}

// TestServeHTTP_issueNonUpdateActionsNoUpdatedExtensions locks in that
// create and remove actions do NOT emit the updated<field> extensions —
// those are scoped to update events. The payloads include a stray
// updatedFrom as defence-in-depth: even if Linear sends one on a
// non-update event, the trampoline must ignore it so consumers can rely
// on filtering by action= alone. Covering both actions explicitly guards
// against a future refactor flipping the gate to e.g. action != "create"
// and silently regressing the remove path.
func TestServeHTTP_issueNonUpdateActionsNoUpdatedExtensions(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := clockwork.NewFakeClockAt(now)

	tests := []struct {
		name     string
		action   string
		delivery string
	}{
		{name: "create", action: "create", delivery: "delivery-create"},
		{name: "remove", action: "remove", delivery: "delivery-remove"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := &fakeClient{}
			s := NewServer(client, [][]byte{[]byte(testSecret)})
			s.clock = clock

			body := fmt.Sprintf(
				`{"action":"%s","type":"Issue","organizationId":"org-123","webhookId":"wh-456","webhookTimestamp":%d,"createdAt":"2025-01-01T00:00:00.000Z","url":"https://linear.app/team/issue/ENG-1","data":{"id":"issue-789","team":{"key":"ENG"}},"updatedFrom":{"description":"shouldnt-leak"}}`,
				tc.action,
				now.UnixMilli(),
			)
			sig := sign(body)

			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
			req.Header.Set("Linear-Signature", sig)
			req.Header.Set("Linear-Event", "Issue")
			req.Header.Set("Linear-Delivery", tc.delivery)

			w := httptest.NewRecorder()
			s.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("w.Code = %d, want %d", w.Code, http.StatusOK)
			}
			if len(client.events) != 1 {
				t.Fatalf("len(client.events) = %d, want 1", len(client.events))
			}
			if _, present := client.events[0].Extensions()["updateddescription"]; present {
				t.Errorf("updateddescription must not be set on action=%s events", tc.action)
			}
		})
	}
}

func TestServeHTTP_commentExtensions(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := clockwork.NewFakeClockAt(now)
	client := &fakeClient{}

	s := NewServer(client, [][]byte{[]byte(testSecret)})
	s.clock = clock

	body := newCommentPayload("create", now.UnixMilli())
	sig := sign(body)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Linear-Signature", sig)
	req.Header.Set("Linear-Event", "Comment")
	req.Header.Set("Linear-Delivery", "delivery-comment-1")

	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("w.Code = %d, want %d: %s", w.Code, http.StatusOK, w.Body.String())
	}
	if len(client.events) != 1 {
		t.Fatalf("len(client.events) = %d, want 1", len(client.events))
	}

	event := client.events[0]
	if got, want := event.Type(), "dev.chainguard.linear.comment"; got != want {
		t.Errorf("type = %s, want %s", got, want)
	}

	// Comment events should set issueid to the parent issue ID.
	issueURL, _ := event.Extensions()["issueid"].(string)
	if issueURL != "issue-789" {
		t.Errorf("issueid = %s, want issue-789", issueURL)
	}

	// Comment events should set the team extension extracted from the URL.
	team, _ := event.Extensions()["team"].(string)
	if team != "ENG" {
		t.Errorf("team = %s, want ENG", team)
	}

	// Comment events should set authorid (Linear user UUID) so downstream
	// subscribers can use cloudevent-trigger's filter_not to skip comments
	// from automation bots, self-loops, etc. authorname is the
	// human-readable companion for log/debug contexts.
	authorID, _ := event.Extensions()["authorid"].(string)
	if authorID != "user-1" {
		t.Errorf("authorid = %q, want %q", authorID, "user-1")
	}
	authorName, _ := event.Extensions()["authorname"].(string)
	if authorName != "Auto Bot" {
		t.Errorf("authorname = %q, want %q", authorName, "Auto Bot")
	}
}

func TestServeHTTP_nonIssueEventExtensions(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := clockwork.NewFakeClockAt(now)
	client := &fakeClient{}

	s := NewServer(client, [][]byte{[]byte(testSecret)})
	s.clock = clock

	// Project events should NOT set issueid or team extensions.
	body := newTestPayload("create", "Project", now.UnixMilli())
	sig := sign(body)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Linear-Signature", sig)
	req.Header.Set("Linear-Event", "Project")
	req.Header.Set("Linear-Delivery", "delivery-project-1")

	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("w.Code = %d, want %d: %s", w.Code, http.StatusOK, w.Body.String())
	}
	if len(client.events) != 1 {
		t.Fatalf("len(client.events) = %d, want 1", len(client.events))
	}

	event := client.events[0]
	if _, ok := event.Extensions()["issueid"]; ok {
		t.Errorf("project event should not have issueid extension")
	}
	if _, ok := event.Extensions()["team"]; ok {
		t.Errorf("project event should not have team extension")
	}
	// authorid/authorname are Comment-only — Project events must not set them
	// even if a payload happens to include a top-level actor record. Guards
	// against accidentally widening the scope to non-Comment events.
	if _, ok := event.Extensions()["authorid"]; ok {
		t.Errorf("project event should not have authorid extension")
	}
	if _, ok := event.Extensions()["authorname"]; ok {
		t.Errorf("project event should not have authorname extension")
	}
}

func TestTeamKeyFromURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{"issue URL", "https://linear.app/chainguard/issue/DEV-747/some-title", "DEV"},
		{"comment URL", "https://linear.app/chainguard/issue/ENG-1/test#comment-abc", "ENG"},
		{"long team key", "https://linear.app/chainguard/issue/LIBRQ-182/title", "LIBRQ"},
		{"query suffix", "https://linear.app/chainguard/issue/DEV-1?foo=bar", "DEV"},
		{"first match wins", "https://linear.app/x/issue/AAA-1/issue/BBB-2", "AAA"},
		{"no match", "https://linear.app/chainguard/project/foo", ""},
		{"empty string", "", ""},
		// Lowercase keys are rejected — Linear team keys are uppercase.
		{"lowercase rejected", "https://linear.app/chainguard/issue/dev-1/title", ""},
		// Bound the team key to reject pathological inputs.
		{"team key too long", "https://linear.app/chainguard/issue/AAAAAAAAAAA-1/title", ""},
		// Trailing junk must not silently match a different identifier.
		{"trailing junk on number", "https://linear.app/chainguard/issue/ABC-12extra", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := teamKeyFromURL(tc.url); got != tc.want {
				t.Errorf("teamKeyFromURL(%q) = %q, want %q", tc.url, got, tc.want)
			}
		})
	}
}

func TestValidateSignature(t *testing.T) {
	body := []byte(`{"test": "data"}`)
	secret := []byte("my-secret")

	mac := hmac.New(sha256.New, secret)
	mac.Write(body)
	validSig := hex.EncodeToString(mac.Sum(nil))

	tests := []struct {
		name      string
		body      []byte
		signature string
		secrets   [][]byte
		want      bool
	}{{
		name:      "valid signature",
		body:      body,
		signature: validSig,
		secrets:   [][]byte{secret},
		want:      true,
	}, {
		name:      "invalid signature",
		body:      body,
		signature: "0000000000000000000000000000000000000000000000000000000000000000",
		secrets:   [][]byte{secret},
		want:      false,
	}, {
		name:      "non-hex signature",
		body:      body,
		signature: "not-hex",
		secrets:   [][]byte{secret},
		want:      false,
	}, {
		name:      "wrong secret",
		body:      body,
		signature: validSig,
		secrets:   [][]byte{[]byte("wrong-secret")},
		want:      false,
	}, {
		name:      "second secret matches",
		body:      body,
		signature: validSig,
		secrets:   [][]byte{[]byte("wrong"), secret},
		want:      true,
	}, {
		name:      "no secrets",
		body:      body,
		signature: validSig,
		secrets:   nil,
		want:      false,
	}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := validateSignature(tc.body, tc.signature, tc.secrets)
			if got != tc.want {
				t.Errorf("validateSignature() = %v, want %v", got, tc.want)
			}
		})
	}
}
