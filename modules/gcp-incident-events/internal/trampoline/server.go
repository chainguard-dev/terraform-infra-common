/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package trampoline

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/chainguard-dev/clog"
	cloudevents "github.com/cloudevents/sdk-go/v2"
)

// EventType is the CloudEvent type of every forwarded notification, opens
// and closes alike.
const EventType = "dev.chainguard.gcp.monitoring.incident"

// maxBodyBytes is the Pub/Sub maximum message size.
const maxBodyBytes = 10 << 20

// messageIDHeader carries the Pub/Sub message ID when the push subscription
// uses no_wrapper with write_metadata.
const messageIDHeader = "X-Goog-Pubsub-Message-Id"

// Server forwards Cloud Monitoring incident notifications to the broker
// ingress as CloudEvents.
type Server struct {
	client cloudevents.Client
}

// NewServer creates a Server that sends events with the given client.
func NewServer(client cloudevents.Client) *Server {
	return &Server{client: client}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	if err != nil {
		clog.ErrorContextf(ctx, "failed to read body: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var n notification
	if err := json.Unmarshal(body, &n); err != nil {
		clog.ErrorContextf(ctx, "failed to unmarshal notification: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if n.Incident.IncidentID == "" {
		clog.ErrorContextf(ctx, "notification has no incident_id")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ctx = clog.WithValues(ctx,
		"incident-id", n.Incident.IncidentID,
		"state", n.Incident.State,
		"policy-name", n.Incident.PolicyName,
	)

	event := cloudevents.NewEvent()
	if id := r.Header.Get(messageIDHeader); id != "" {
		event.SetID(id)
	}
	event.SetType(EventType)
	event.SetSource(r.Host)
	event.SetSubject(n.Incident.IncidentID)
	for name, value := range map[string]string{
		"state":        n.Incident.State,
		"project":      n.Incident.ScopingProjectID,
		"resourcetype": n.Incident.Resource.Type,
		"policyname":   n.Incident.PolicyName,
		"severity":     n.Incident.Severity,
	} {
		if value != "" {
			event.SetExtension(name, value)
		}
	}
	if err := event.SetData(cloudevents.ApplicationJSON, body); err != nil {
		clog.ErrorContextf(ctx, "failed to set data: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	const retryDelay = 10 * time.Millisecond
	const maxRetry = 3
	rctx := cloudevents.ContextWithRetriesExponentialBackoff(context.WithoutCancel(ctx), retryDelay, maxRetry)
	if ceresult := s.client.Send(rctx, event); cloudevents.IsUndelivered(ceresult) || cloudevents.IsNACK(ceresult) {
		clog.ErrorContextf(ctx, "failed to deliver event: %v", ceresult)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	clog.DebugContextf(ctx, "event forwarded")
}

// notification holds the fields of a Cloud Monitoring Pub/Sub notification
// that become CloudEvent attributes. The event data is the original body, so
// fields not listed here still reach consumers.
type notification struct {
	Incident struct {
		IncidentID       string `json:"incident_id"`
		State            string `json:"state"`
		ScopingProjectID string `json:"scoping_project_id"`
		PolicyName       string `json:"policy_name"`
		Severity         string `json:"severity"`
		Resource         struct {
			Type string `json:"type"`
		} `json:"resource"`
	} `json:"incident"`
}
