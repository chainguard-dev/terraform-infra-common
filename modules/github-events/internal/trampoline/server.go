package trampoline

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"time"

	"github.com/chainguard-dev/clog"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/go-github/v61/github"
	"github.com/jonboulle/clockwork"
)

type Server struct {
	client  cloudevents.Client
	secrets [][]byte
	clock   clockwork.Clock
}

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

	// https://docs.github.com/en/webhooks/using-webhooks/validating-webhook-deliveries
	payload, err := ValidatePayload(r, s.secrets)
	if err != nil {
		log.Errorf("failed to verify webhook: %v", err)
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprintf(w, "failed to verify webhook: %v", err)
		return
	}

	// https://docs.github.com/en/webhooks/webhook-events-and-payloads#delivery-headers
	t := github.WebHookType(r)
	if t == "" {
		log.Errorf("missing X-GitHub-Event header")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	t = "dev.chainguard.github." + t
	log = log.With("event-type", t)

	var msg struct {
		Action     string `json:"action"`
		Repository struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
	}
	if err := json.Unmarshal(payload, &msg); err != nil {
		log.Warnf("failed to unmarshal payload; action and subject will be unset: %v", err)
	} else {
		log = log.With("action", msg.Action, "repo", msg.Repository.FullName)
	}

	log.Debugf("forwarding event: %s", t)

	event := cloudevents.NewEvent()
	event.SetID(github.DeliveryID(r))
	event.SetType(t)
	event.SetSource(r.Host)
	event.SetSubject(msg.Repository.FullName)
	event.SetExtension("action", msg.Action)
	// Needs to be an extension to be a filterable attribute.
	// See https://github.com/chainguard-dev/terraform-infra-common/blob/main/pkg/pubsub/cloudevent.go
	if id := r.Header.Get("X-GitHub-Hook-ID"); id != "" {
		// Cloud Event attribute spec only allows [a-z0-9] :(
		event.SetExtension("githubhook", id)
	}
	if err := event.SetData(cloudevents.ApplicationJSON, eventData{
		When: s.clock.Now(),
		Headers: &eventHeaders{
			HookID:                 r.Header.Get("X-GitHub-Hook-ID"),
			DeliveryID:             r.Header.Get("X-GitHub-Delivery"),
			UserAgent:              r.Header.Get("User-Agent"),
			Event:                  r.Header.Get("X-GitHub-Event"),
			InstallationTargetType: r.Header.Get("X-GitHub-Installation-Target-Type"),
			InstallationTargetID:   r.Header.Get("X-GitHub-Installation-Target-ID"),
		},
		Body: payload,
	}); err != nil {
		log.Errorf("failed to set data: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	const retryDelay = 10 * time.Millisecond
	const maxRetry = 3
	rctx := cloudevents.ContextWithRetriesExponentialBackoff(context.WithoutCancel(ctx), retryDelay, maxRetry)
	if ceresult := s.client.Send(rctx, event); cloudevents.IsUndelivered(ceresult) || cloudevents.IsNACK(ceresult) {
		log.Errorf("Failed to deliver event: %v", ceresult)
		w.WriteHeader(http.StatusInternalServerError)
	}
	log.Debugf("event forwarded")
}

type eventData struct {
	When time.Time `json:"when"`
	// See https://docs.github.com/en/webhooks/webhook-events-and-payloads#delivery-headers
	Headers *eventHeaders   `json:"headers,omitempty"`
	Body    json.RawMessage `json:"body"`
}

// Relevant headers for GitHub webhook events that we want to record.
// See https://docs.github.com/en/webhooks/webhook-events-and-payloads#delivery-headers
type eventHeaders struct {
	HookID                 string `json:"hook_id,omitempty"`
	DeliveryID             string `json:"delivery_id,omitempty"`
	UserAgent              string `json:"user_agent,omitempty"`
	Event                  string `json:"event,omitempty"`
	InstallationTargetType string `json:"installation_target_type,omitempty"`
	InstallationTargetID   string `json:"installation_target_id,omitempty"`
}

// ValidatePayload validates the payload of a webhook request for a given set of secrets.
// If any of the secrets are valid, the payload is returned with no error.
func ValidatePayload(r *http.Request, secrets [][]byte) ([]byte, error) {
	// Largely forked from github.ValidatePayload - we can't use this directly to avoid consuming the body.
	signature := r.Header.Get(github.SHA256SignatureHeader)
	if signature == "" {
		signature = r.Header.Get(github.SHA1SignatureHeader)
	}
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	for _, secret := range secrets {
		payload, err := github.ValidatePayloadFromBody(contentType, bytes.NewBuffer(body), signature, secret)
		if err == nil {
			return payload, nil
		}
	}
	return nil, fmt.Errorf("failed to validate payload")
}
