/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package trampoline

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/go-cmp/cmp"
)

type fakeClient struct {
	cloudevents.Client
	result cloudevents.Result
	events []cloudevents.Event
}

func (f *fakeClient) Send(_ context.Context, event cloudevents.Event) cloudevents.Result {
	f.events = append(f.events, event)
	return f.result
}

// cloudRunOpen is shaped like an incident from a log-based policy on a Cloud
// Run service: the service name is in the resource labels.
func cloudRunOpen(incidentID, project string) string {
	return fmt.Sprintf(`{"incident":{"incident_id":%q,"scoping_project_id":%q,"scoping_project_number":12345,"severity":"Error","started_at":1577840461,"ended_at":null,"state":"open","resource":{"type":"cloud_run_revision","labels":{"project_id":%q,"service_name":"svc-a","location":"us-central1"}},"policy_name":"Panic log entry","policy_user_labels":{"team":"sus-tools"},"documentation":{"content":"Check the logs.","mime_type":"text/markdown","subject":"Panic"},"summary":"Panic in svc-a."},"version":"1.2"}`, incidentID, project, project)
}

// prometheusClosed is shaped like the workqueue dead-letter incident: the
// resource has only project_id and the service name is a metric label.
func prometheusClosed(incidentID, project string) string {
	return fmt.Sprintf(`{"incident":{"incident_id":%q,"scoping_project_id":%q,"started_at":1577840461,"ended_at":1577877071,"state":"closed","resource":{"type":"prometheus_target","labels":{"project_id":%q}},"metric":{"type":"prometheus.googleapis.com/workqueue_dead_lettered_keys/gauge","labels":{"service_name":"svc-wq-dsp"}},"policy_name":"Workqueue dead-lettered keys"},"version":"1.2"}`, incidentID, project, project)
}

func TestServeHTTP_forwards(t *testing.T) {
	incidentID := rand.Text()
	project := strings.ToLower(rand.Text())
	messageID := rand.Text()

	tests := []struct {
		name           string
		body           string
		messageID      string
		wantID         string
		wantExtensions map[string]any
	}{{
		name:      "open incident with resource labels",
		body:      cloudRunOpen(incidentID, project),
		messageID: messageID,
		wantID:    messageID,
		wantExtensions: map[string]any{
			"state":        "open",
			"project":      project,
			"resourcetype": "cloud_run_revision",
			"policyname":   "Panic log entry",
			"severity":     "Error",
		},
	}, {
		name:      "closed incident with metric labels and no severity",
		body:      prometheusClosed(incidentID, project),
		messageID: messageID,
		wantID:    messageID,
		wantExtensions: map[string]any{
			"state":        "closed",
			"project":      project,
			"resourcetype": "prometheus_target",
			"policyname":   "Workqueue dead-lettered keys",
		},
	}, {
		name: "no message ID leaves the ID to the client",
		body: cloudRunOpen(incidentID, project),
		wantExtensions: map[string]any{
			"state":        "open",
			"project":      project,
			"resourcetype": "cloud_run_revision",
			"policyname":   "Panic log entry",
			"severity":     "Error",
		},
	}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := &fakeClient{}
			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", strings.NewReader(tc.body))
			if tc.messageID != "" {
				req.Header.Set(messageIDHeader, tc.messageID)
			}
			w := httptest.NewRecorder()

			NewServer(client).ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("status: got = %d, want = %d", w.Code, http.StatusOK)
			}
			if len(client.events) != 1 {
				t.Fatalf("events: got = %d, want = 1", len(client.events))
			}
			event := client.events[0]
			if got, want := event.Type(), EventType; got != want {
				t.Errorf("type: got = %q, want = %q", got, want)
			}
			if got, want := event.ID(), tc.wantID; got != want {
				t.Errorf("id: got = %q, want = %q", got, want)
			}
			if got, want := event.Subject(), incidentID; got != want {
				t.Errorf("subject: got = %q, want = %q", got, want)
			}
			if got, want := event.DataContentType(), cloudevents.ApplicationJSON; got != want {
				t.Errorf("datacontenttype: got = %q, want = %q", got, want)
			}
			if diff := cmp.Diff(tc.wantExtensions, event.Extensions()); diff != "" {
				t.Errorf("extensions (-want, +got):\n%s", diff)
			}
			if got, want := string(event.Data()), tc.body; got != want {
				t.Errorf("data: got = %s, want = %s", got, want)
			}
		})
	}
}

func TestServeHTTP_rejects(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		result     cloudevents.Result
		wantStatus int
		wantSent   int
	}{{
		name:       "malformed JSON",
		body:       `{"incident":`,
		wantStatus: http.StatusBadRequest,
	}, {
		name:       "missing incident ID",
		body:       `{"incident":{"state":"open","policy_name":"p"},"version":"1.2"}`,
		wantStatus: http.StatusBadRequest,
	}, {
		name:       "body over the Pub/Sub message limit",
		body:       `{"incident":{"incident_id":"x","summary":"` + strings.Repeat("a", maxBodyBytes) + `"}}`,
		wantStatus: http.StatusBadRequest,
	}, {
		name:       "broker does not accept the event",
		body:       cloudRunOpen(rand.Text(), "p"),
		result:     errors.New("connection refused"),
		wantStatus: http.StatusInternalServerError,
		wantSent:   1,
	}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := &fakeClient{result: tc.result}
			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", strings.NewReader(tc.body))
			w := httptest.NewRecorder()

			NewServer(client).ServeHTTP(w, req)

			if got, want := w.Code, tc.wantStatus; got != want {
				t.Errorf("status: got = %d, want = %d", got, want)
			}
			if got, want := len(client.events), tc.wantSent; got != want {
				t.Errorf("events: got = %d, want = %d", got, want)
			}
		})
	}
}
