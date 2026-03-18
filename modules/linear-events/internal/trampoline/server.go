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
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/chainguard-dev/clog"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/jonboulle/clockwork"
)

// maxTimestampAge is the maximum age of a webhook timestamp before it is
// considered a replay attack.
const maxTimestampAge = 5 * time.Minute

// Server handles incoming Linear webhook requests, validates their signatures,
// converts them to CloudEvents, and forwards them to the broker ingress.
type Server struct {
	client  cloudevents.Client
	secrets [][]byte
	clock   clockwork.Clock
}

// NewServer creates a new Server with the given CloudEvents client and webhook secrets.
func NewServer(client cloudevents.Client, secrets [][]byte) *Server {
	return &Server{
		client:  client,
		secrets: secrets,
		clock:   clockwork.NewRealClock(),
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := clog.FromContext(ctx)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Errorf("failed to read body: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Validate the webhook signature.
	signature := r.Header.Get("Linear-Signature")
	if signature == "" {
		log.Errorf("missing Linear-Signature header")
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, "missing Linear-Signature header")
		return
	}

	if !validateSignature(body, signature, s.secrets) {
		log.Errorf("invalid webhook signature")
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, "invalid webhook signature")
		return
	}

	// Extract Linear headers.
	eventType := r.Header.Get("Linear-Event")
	if eventType == "" {
		log.Errorf("missing Linear-Event header")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	deliveryID := r.Header.Get("Linear-Delivery")

	log = log.With("event-type", eventType, "delivery-id", deliveryID)

	// Parse the body to extract metadata and validate timestamp.
	var payload webhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		log.Errorf("failed to unmarshal payload: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Validate webhook timestamp for replay protection.
	if payload.WebhookTimestamp > 0 {
		webhookTime := time.UnixMilli(payload.WebhookTimestamp)
		if diff := s.clock.Now().Sub(webhookTime); diff < 0 || diff > maxTimestampAge {
			log.Errorf("webhook timestamp too old or in the future: %v (diff: %v)", webhookTime, diff)
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprint(w, "webhook timestamp out of range")
			return
		}
	}

	log = log.With("action", payload.Action)

	// Build the CloudEvent.
	ceType := "dev.chainguard.linear." + strings.ToLower(eventType)
	log.Debugf("forwarding event: %s", ceType)

	event := cloudevents.NewEvent()
	event.SetID(deliveryID)
	event.SetType(ceType)
	event.SetSource(r.Host)
	event.SetSubject(payload.OrganizationID)
	event.SetExtension("action", payload.Action)
	if payload.WebhookID != "" {
		event.SetExtension("webhookid", payload.WebhookID)
	}

	// Set entity-specific extensions based on event type for downstream
	// filtering and workqueue key extraction.
	switch strings.ToLower(eventType) {
	case "issue":
		// Use the immutable UUID as the workqueue key rather than the
		// Linear app URL, which can change if the team slug or workspace
		// is renamed.
		if payload.Data.ID != "" {
			event.SetExtension("issueid", payload.Data.ID)
		}
		if payload.Data.Team.Key != "" {
			event.SetExtension("team", payload.Data.Team.Key)
		}
	case "comment":
		if payload.Data.IssueID != "" {
			event.SetExtension("issueid", payload.Data.IssueID)
		}
	}

	if err := event.SetData(cloudevents.ApplicationJSON, eventData{
		When: s.clock.Now(),
		Headers: &eventHeaders{
			DeliveryID: deliveryID,
			Event:      eventType,
			WebhookID:  payload.WebhookID,
		},
		Body: body,
	}); err != nil {
		log.Errorf("failed to set data: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	const retryDelay = 10 * time.Millisecond
	const maxRetry = 3
	rctx := cloudevents.ContextWithRetriesExponentialBackoff(context.WithoutCancel(ctx), retryDelay, maxRetry)
	if ceresult := s.client.Send(rctx, event); cloudevents.IsUndelivered(ceresult) || cloudevents.IsNACK(ceresult) {
		log.Errorf("failed to deliver event: %v", ceresult)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	log.Debugf("event forwarded")
}

// webhookPayload captures the top-level fields of a Linear webhook payload
// needed for processing and validation.
type webhookPayload struct {
	Action           string `json:"action"`
	Type             string `json:"type"`
	OrganizationID   string `json:"organizationId"`
	WebhookID        string `json:"webhookId"`
	WebhookTimestamp int64  `json:"webhookTimestamp"`

	Data webhookData `json:"data"`
}

// webhookData captures entity-specific fields from the webhook payload
// used to set CloudEvent extensions for downstream filtering and routing.
type webhookData struct {
	ID      string      `json:"id"`
	IssueID string      `json:"issueId"` // set on Comment events
	Team    webhookTeam `json:"team"`
}

type webhookTeam struct {
	Key string `json:"key"`
}

type eventData struct {
	When    time.Time       `json:"when"`
	Headers *eventHeaders   `json:"headers,omitempty"`
	Body    json.RawMessage `json:"body"`
}

type eventHeaders struct {
	DeliveryID string `json:"delivery_id,omitempty"`
	Event      string `json:"event,omitempty"`
	WebhookID  string `json:"webhook_id,omitempty"`
}

// validateSignature checks the HMAC-SHA256 signature of the raw body against
// any of the provided secrets. Returns true if any secret produces a matching
// signature.
func validateSignature(body []byte, signature string, secrets [][]byte) bool {
	sigBytes, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}

	for _, secret := range secrets {
		mac := hmac.New(sha256.New, secret)
		mac.Write(body)
		expected := mac.Sum(nil)
		if hmac.Equal(sigBytes, expected) {
			return true
		}
	}
	return false
}
