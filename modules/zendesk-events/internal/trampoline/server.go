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
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/chainguard-dev/clog"
	"github.com/chainguard-dev/terraform-infra-common/modules/zendesk-events/internal/redact"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/jonboulle/clockwork"
)

// maxTimestampAge is the maximum age of a webhook timestamp before it is
// considered a replay attack.
const maxTimestampAge = 5 * time.Minute

// maxBodyBytes caps the request body read before HMAC verification, so an
// unauthenticated peer cannot exhaust memory with a large POST. Zendesk
// webhook bodies are well under this in practice.
const maxBodyBytes = 1 << 20 // 1 MiB

const (
	// signatureHeader carries the base64-encoded HMAC-SHA256 signature.
	signatureHeader = "X-Zendesk-Webhook-Signature"
	// signatureTimestampHeader carries the timestamp that is prepended to the
	// request body before computing the signature. Zendesk signs
	// timestamp+body, NOT the body alone.
	signatureTimestampHeader = "X-Zendesk-Webhook-Signature-Timestamp"

	// eventTypePrefix is stripped from the payload `type` field to derive the
	// CloudEvent type suffix. Zendesk event-subscription payloads carry a type
	// like "zen:event-type:ticket.created".
	eventTypePrefix = "zen:event-type:"
	// ceTypePrefix is prepended to the derived suffix to form the CloudEvent
	// type, e.g. "dev.chainguard.zendesk.ticket.created".
	ceTypePrefix = "dev.chainguard.zendesk."

	// ticketSubjectPrefix prefixes the `subject` field for ticket events, e.g.
	// "zen:ticket:35436".
	ticketSubjectPrefix = "zen:ticket:"

	// eventSource is the constant CloudEvent source for events this trampoline
	// emits. The ingress hostname is attacker-controllable (the Cloud Run
	// service accepts INGRESS_TRAFFIC_ALL by default), so we don't use it.
	eventSource = "zendesk-trampoline"
)

// Server handles incoming Zendesk webhook requests, validates their signatures,
// converts them to CloudEvents, and forwards them to the broker ingress.
type Server struct {
	client  cloudevents.Client
	secrets [][]byte
	clock   clockwork.Clock
}

// placeholderSecret is the constant value the Terraform ../secret module writes
// as the first secret version when create_placeholder_version is set, before an
// operator uploads the real Zendesk signing secret. It is public in this repo,
// so it must never be accepted as a usable signing key.
const placeholderSecret = "placeholder"

// NewServer creates a new Server with the given CloudEvents client and webhook
// secrets. Any secret equal to the public placeholder value is discarded: the
// pipeline is fail-closed (all requests 403) until a real Zendesk signing
// secret is installed, so the known placeholder cannot be used to forge
// signatures during the bootstrap window.
func NewServer(client cloudevents.Client, secrets [][]byte) *Server {
	usable := make([][]byte, 0, len(secrets))
	for _, s := range secrets {
		if string(s) == placeholderSecret {
			slog.Warn("trampoline: ignoring placeholder webhook secret; requests will be rejected until a real Zendesk signing secret is installed")
			continue
		}
		usable = append(usable, s)
	}
	if len(usable) == 0 {
		slog.Warn("trampoline: no usable webhook secret configured; all requests will be rejected")
	}
	return &Server{
		client:  client,
		secrets: usable,
		clock:   clockwork.NewRealClock(),
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := clog.FromContext(ctx)

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	if err != nil {
		log.Errorf("failed to read body: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Validate the webhook signature. Zendesk computes the HMAC over the
	// timestamp header concatenated with the raw body, so both headers are
	// required to verify the request.
	signature := r.Header.Get(signatureHeader)
	if signature == "" {
		log.Errorf("missing %s header", signatureHeader)
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprintf(w, "missing %s header", signatureHeader)
		return
	}
	timestamp := r.Header.Get(signatureTimestampHeader)
	if timestamp == "" {
		log.Errorf("missing %s header", signatureTimestampHeader)
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprintf(w, "missing %s header", signatureTimestampHeader)
		return
	}

	if !validateSignature(body, timestamp, signature, s.secrets) {
		log.Errorf("invalid webhook signature")
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, "invalid webhook signature")
		return
	}

	// Validate the signature timestamp for replay protection. The timestamp is
	// part of the signed message, so an attacker cannot forge a fresh one
	// without the secret — but rejecting stale timestamps bounds the replay
	// window for a captured-and-replayed request.
	if t, err := time.Parse(time.RFC3339, timestamp); err != nil {
		log.Errorf("failed to parse %s %q: %v", signatureTimestampHeader, timestamp, err)
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, "invalid webhook timestamp")
		return
	} else if diff := s.clock.Now().Sub(t); diff < -maxTimestampAge || diff > maxTimestampAge {
		log.Errorf("webhook timestamp out of range: %v (diff: %v)", t, diff)
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, "webhook timestamp out of range")
		return
	}

	// Parse the body to extract the event type and metadata.
	var payload webhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		log.Errorf("failed to unmarshal payload: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Derive the CloudEvent type from the payload `type` field. Zendesk event
	// subscriptions carry a type like "zen:event-type:ticket.created"; we
	// require that prefix so trigger/automation webhooks (which carry a custom
	// body without it) are rejected rather than silently mis-typed.
	if !strings.HasPrefix(payload.Type, eventTypePrefix) {
		log.Errorf("payload type %q missing %q prefix", payload.Type, eventTypePrefix)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "unsupported or missing event type")
		return
	}
	suffix := strings.ToLower(strings.TrimPrefix(payload.Type, eventTypePrefix))
	ceType := ceTypePrefix + suffix

	// Forward ticket.* events only. The recorder ships schemas for ticket.*
	// exclusively, and the redaction allowlist is only safe for ticket events:
	// generic keys like previous/current carry status/priority enums for
	// ticket.status_changed / ticket.priority_changed, but on other resources
	// (e.g. organization.name_changed) they carry the customer org name — free
	// text the regex pass would not catch. Rejecting non-ticket resources keeps
	// that latent PII out of the broker / GCS / BigQuery.
	resource := suffix
	if i := strings.IndexByte(suffix, '.'); i >= 0 {
		resource = suffix[:i]
	}
	if resource != "ticket" {
		log.Errorf("unsupported event resource %q (only ticket.* events are forwarded)", resource)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "unsupported event type")
		return
	}

	log = log.With("event-type", payload.Type, "event-id", payload.ID)
	log.Debugf("forwarding event: %s", ceType)

	event := cloudevents.NewEvent()
	event.SetID(payload.ID)
	event.SetType(ceType)
	event.SetSource(eventSource)
	// Run the subject through the regex pass so that any identifying tokens
	// (emails, IPs, cgr.dev paths) that appear in non-ticket subjects are
	// replaced before the attribute reaches the broker.
	event.SetSubject(redact.String(payload.Subject))
	if payload.AccountID != 0 {
		event.SetExtension("accountid", strconv.FormatInt(payload.AccountID, 10))
	}

	// Use the immutable ticket ID as the workqueue key for downstream
	// reconcilers. It is encoded in the subject as "zen:ticket:<id>".
	if id := ticketIDFromSubject(payload.Subject); id != "" {
		event.SetExtension("ticketid", id)
	}

	// Strip customer-identifying information from the forwarded body before it
	// reaches the broker / GCS / BigQuery. The signature has already been
	// validated against the original bytes above, and downstream consumers
	// re-fetch authoritative ticket data from Zendesk by id, so redaction here
	// affects only what is persisted.
	if err := event.SetData(cloudevents.ApplicationJSON, eventData{
		When: s.clock.Now(),
		Headers: &eventHeaders{
			EventID:            payload.ID,
			EventType:          payload.Type,
			SignatureTimestamp: timestamp,
		},
		Body: redact.Body(body),
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

// webhookPayload captures the top-level fields of a Zendesk event-subscription
// webhook payload needed for processing. Entity-specific data is forwarded
// untouched in the event body, so only the routing fields are decoded here.
type webhookPayload struct {
	AccountID int64  `json:"account_id"`
	ID        string `json:"id"`
	Subject   string `json:"subject"`
	Type      string `json:"type"`
	Time      string `json:"time"`
}

type eventData struct {
	When    time.Time       `json:"when"`
	Headers *eventHeaders   `json:"headers,omitempty"`
	Body    json.RawMessage `json:"body"`
}

type eventHeaders struct {
	EventID            string `json:"event_id,omitempty"`
	EventType          string `json:"event_type,omitempty"`
	SignatureTimestamp string `json:"signature_timestamp,omitempty"`
}

// ticketIDFromSubject extracts the numeric ticket ID from a Zendesk ticket
// subject ("zen:ticket:<id>"). Returns empty string when the subject is not a
// ticket subject; callers treat that as "skip the extension".
func ticketIDFromSubject(subject string) string {
	if !strings.HasPrefix(subject, ticketSubjectPrefix) {
		return ""
	}
	return strings.TrimPrefix(subject, ticketSubjectPrefix)
}

// validateSignature checks the base64-encoded HMAC-SHA256 signature of
// (timestamp + body) against any of the provided secrets. Returns true if any
// secret produces a matching signature.
func validateSignature(body []byte, timestamp, signature string, secrets [][]byte) bool {
	sigBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return false
	}

	// Zendesk signs the concatenation of the timestamp header and the raw body.
	msg := make([]byte, 0, len(timestamp)+len(body))
	msg = append(msg, timestamp...)
	msg = append(msg, body...)

	for _, secret := range secrets {
		mac := hmac.New(sha256.New, secret)
		mac.Write(msg)
		expected := mac.Sum(nil)
		if hmac.Equal(sigBytes, expected) {
			return true
		}
	}
	return false
}
