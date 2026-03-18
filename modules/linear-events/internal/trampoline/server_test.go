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
	return fmt.Sprintf(`{"action":%q,"type":"Comment","organizationId":"org-123","webhookId":"wh-456","webhookTimestamp":%d,"data":{"id":"comment-001","body":"A comment","issueId":"issue-789","userId":"user-1"}}`, action, ts)
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

	// Comment events should NOT set the team extension.
	if _, ok := event.Extensions()["team"]; ok {
		t.Errorf("comment event should not have team extension")
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
