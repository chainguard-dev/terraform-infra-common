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
	"github.com/google/go-github/v72/github"
	"github.com/jonboulle/clockwork"
)

// PayloadInfo is a minimal struct for GitHub webhook payload information,
// containing only the fields we need to process for our needs of setting cloud event headers.
type PayloadInfo struct {
	Action     string `json:"action,omitempty"`
	Number     int    `json:"number,omitempty"`
	Repository struct {
		FullName string `json:"full_name,omitempty"`
		Owner    struct {
			Login string `json:"login,omitempty"`
		} `json:"owner,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"repository,omitempty"`
	Organization struct {
		Login string `json:"login,omitempty"`
	} `json:"organization,omitempty"`
	PullRequest struct {
		Number int  `json:"number,omitempty"`
		Merged bool `json:"merged,omitempty"`
	} `json:"pull_request,omitempty"`
	Issue struct {
		Number          int       `json:"number,omitempty"`
		PullRequestInfo *struct{} `json:"pull_request,omitempty"`
	} `json:"issue,omitempty"`
	CheckRun struct {
		PullRequests []struct {
			Number int `json:"number,omitempty"`
		} `json:"pull_requests,omitempty"`
	} `json:"check_run,omitempty"`
	CheckSuite struct {
		PullRequests []struct {
			Number int `json:"number,omitempty"`
		} `json:"pull_requests,omitempty"`
	} `json:"check_suite,omitempty"`
	Comment struct {
		ID int `json:"id,omitempty"`
	} `json:"comment,omitempty"`
	Review struct {
		ID int `json:"id,omitempty"`
	} `json:"review,omitempty"`
}

type Server struct {
	client  cloudevents.Client
	secrets [][]byte
	clock   clockwork.Clock
	// webhookID is an optional config that will instruct the trampoline to only listen to events coming from a specific webhook.
	// If webhookID is empty, the trampoline will listen to all events.
	webhookID            []string
	requestedOnlyWebhook []string
	orgFilter            []string
}

type ServerOptions struct {
	Secrets              [][]byte
	WebhookID            []string
	RequestedOnlyWebhook []string
	OrgFilter            []string
}

func NewServer(client cloudevents.Client, opts ServerOptions) *Server {
	return &Server{
		client:               client,
		secrets:              opts.Secrets,
		requestedOnlyWebhook: opts.RequestedOnlyWebhook,
		webhookID:            opts.WebhookID,
		orgFilter:            opts.OrgFilter,
		clock:                clockwork.NewRealClock(),
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
	log = log.With("event-type", t)

	hookID := r.Header.Get("X-GitHub-Hook-ID")
	// If webhookID is set, only listen to events from the specified webhook.
	if len(s.webhookID) > 0 {
		found := false
		for _, id := range s.webhookID {
			if hookID == id {
				found = true
				break
			}
		}
		if !found {
			log.Warnf("ignoring event from webhook due to webhook_id %q %q", hookID, github.DeliveryID(r))
			// Use 202 Accepted to as an ACK, but no action taken.
			w.WriteHeader(http.StatusAccepted)
			return
		}
	}

	// Unmarshal payload to extract necessary information
	var info PayloadInfo
	if err := json.Unmarshal(payload, &info); err != nil {
		log.Warnf("failed to unmarshal payload, cloud event headers will not be set: %v", err)
	}

	// If requestedOnlyWebhook is set, only listen to events from the specified webhook if the event is a requested event.
	var requested bool
	if t == "check_run" || t == "check_suite" {
		requested = info.Action == "requested" || info.Action == "rerequested" || info.Action == "requested_action"
	}
	for _, id := range s.requestedOnlyWebhook {
		if !requested && hookID == id {
			log.Warnf("ignoring event from webhook due to non-requested event %q %q", hookID, github.DeliveryID(r))
			// Use 202 Accepted to as an ACK, but no action taken.
			w.WriteHeader(http.StatusAccepted)
			return
		}
	}

	// Store original event type for extension extraction
	originalEventType := t
	t = "dev.chainguard.github." + t

	// Extract repository and organization information
	repoFullName := info.Repository.FullName
	orgLogin := info.Organization.Login

	log = log.With("action", info.Action, "repo", repoFullName)

	// Filter webhook at org level.
	if len(s.orgFilter) > 0 {
		found := false
		for _, org := range s.orgFilter {
			if orgLogin == org {
				found = true
				break
			}
		}
		if !found {
			log.Warnf("ignoring event from repository %q due to non-matching org", repoFullName)
			w.WriteHeader(http.StatusAccepted)
			return
		}
	}

	log.Debugf("forwarding event: %s", t)

	event := cloudevents.NewEvent()
	event.SetID(github.DeliveryID(r))
	event.SetType(t)
	event.SetSource(r.Host)
	event.SetSubject(repoFullName)
	event.SetExtension("action", info.Action)
	// Needs to be an extension to be a filterable attribute.
	// See https://github.com/chainguard-dev/terraform-infra-common/blob/main/pkg/pubsub/cloudevent.go
	if hookID != "" {
		// Cloud Event attribute spec only allows [a-z0-9] :(
		event.SetExtension("githubhook", hookID)
	}

	// Add pullrequest extension for pull request events (original format)
	if prInfo := extractPullRequestInfo(originalEventType, info); prInfo != "" {
		event.SetExtension("pullrequest", prInfo)
	}

	// Add pullrequest-url extension for PR-related events
	if prURL := extractPullRequestURL(originalEventType, info); prURL != "" {
		event.SetExtension("pullrequesturl", prURL)
	}

	// Add issue-url extension for issue-related events
	if issueURL := extractIssueURL(originalEventType, info); issueURL != "" {
		event.SetExtension("issueurl", issueURL)
	}

	// Add merged extension for merged pull requests
	if merged := isPullRequestMerged(originalEventType, info); merged {
		event.SetExtension("merged", true)
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

// extractPullRequestInfo extracts pull request information from GitHub payload
// Returns a formatted string in the format "org/repo#number" or empty string if not a PR event
func extractPullRequestInfo(eventType string, info PayloadInfo) string {
	// Only process pull_request events
	if eventType != "pull_request" {
		return ""
	}

	// Extract information from our typed struct
	if info.PullRequest.Number > 0 && info.Repository.FullName != "" {
		return fmt.Sprintf("%s#%d", info.Repository.FullName, info.PullRequest.Number)
	}

	return ""
}

// extractPullRequestURL extracts the pull request URL from GitHub events that pertain to a PR
func extractPullRequestURL(eventType string, info PayloadInfo) string {
	owner := info.Repository.Owner.Login
	repo := info.Repository.Name
	if owner == "" || repo == "" {
		return ""
	}

	var prNumber int
	switch eventType {
	case "pull_request":
		prNumber = info.PullRequest.Number
	case "pull_request_review":
		prNumber = info.PullRequest.Number
	case "pull_request_review_comment":
		prNumber = info.PullRequest.Number
	case "issue_comment":
		// Check if this is a PR comment (issue comments can be on PRs too)
		if info.Issue.PullRequestInfo != nil && info.Issue.Number > 0 {
			prNumber = info.Issue.Number
		}
	case "check_run":
		if len(info.CheckRun.PullRequests) > 0 {
			prNumber = info.CheckRun.PullRequests[0].Number
		}
	case "check_suite":
		if len(info.CheckSuite.PullRequests) > 0 {
			prNumber = info.CheckSuite.PullRequests[0].Number
		}
	}

	if prNumber > 0 {
		return fmt.Sprintf("https://github.com/%s/%s/pull/%d", owner, repo, prNumber)
	}
	return ""
}

// extractIssueURL extracts the issue URL from GitHub events that pertain to an issue
func extractIssueURL(eventType string, info PayloadInfo) string {
	owner := info.Repository.Owner.Login
	repo := info.Repository.Name
	if owner == "" || repo == "" {
		return ""
	}

	var issueNumber int
	switch eventType {
	case "issues":
		issueNumber = info.Issue.Number
	case "issue_comment":
		// Only add issue URL if this is NOT a PR comment
		if info.Issue.PullRequestInfo == nil && info.Issue.Number > 0 {
			issueNumber = info.Issue.Number
		}
	}

	if issueNumber > 0 {
		return fmt.Sprintf("https://github.com/%s/%s/issues/%d", owner, repo, issueNumber)
	}
	return ""
}

// isPullRequestMerged checks if a pull request event is for a merged PR
// Returns true if the event is a pull_request with "closed" action and merged=true
func isPullRequestMerged(eventType string, info PayloadInfo) bool {
	// Only process pull_request events
	if eventType != "pull_request" {
		return false
	}

	// A merged PR will have action="closed" and merged=true
	return info.Action == "closed" && info.PullRequest.Merged
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
