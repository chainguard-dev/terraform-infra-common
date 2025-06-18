/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/chainguard-dev/clog"
	_ "github.com/chainguard-dev/clog/gcp/init"
	"github.com/chainguard-dev/terraform-infra-common/pkg/workqueue"
	"github.com/google/go-github/v72/github"
	"github.com/sethvargo/go-envconfig"
)

type Config struct {
	Port             string `env:"PORT,default=8080"`
	WorkqueueService string `env:"WORKQUEUE_SERVICE,required"`
	WebhookSecret    string `env:"GITHUB_WEBHOOK_SECRET,required"`
	ResourceFilter   string `env:"RESOURCE_FILTER"` // Optional: "issues" or "pull_requests"
}

func main() {
	ctx := context.Background()
	logger := clog.FromContext(ctx)

	var cfg Config
	if err := envconfig.Process(ctx, &cfg); err != nil {
		logger.Fatalf("Failed to process configuration: %v", err)
	}

	// Validate resource filter if specified
	if cfg.ResourceFilter != "" && cfg.ResourceFilter != "issues" && cfg.ResourceFilter != "pull_requests" {
		logger.Fatalf("Invalid resource filter %q: must be 'issues' or 'pull_requests'", cfg.ResourceFilter)
	}

	// Log configuration
	logFields := []interface{}{
		"port", cfg.Port,
		"workqueue_service", cfg.WorkqueueService,
	}
	if cfg.ResourceFilter != "" {
		logFields = append(logFields, "resource_filter", cfg.ResourceFilter)
	}
	logger.With(logFields...).Info("Starting GitHub webhook shim")

	// Create workqueue client
	queueClient, err := workqueue.NewWorkqueueClient(ctx, cfg.WorkqueueService)
	if err != nil {
		logger.Fatalf("Failed to create workqueue client: %v", err)
	}
	defer queueClient.Close()

	// Create webhook handler
	handler := &webhookHandler{
		logger:      logger,
		queueClient: queueClient,
		cfg:         cfg,
	}

	// Set up HTTP routes
	http.HandleFunc("/webhook", handler.handleWebhook)

	// Health check endpoint
	http.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("GitHub Webhook Shim\n")); err != nil {
			logger.Errorf("Failed to write response: %v", err)
		}
	})

	logger.With("port", cfg.Port).Info("Starting webhook shim service")

	// Create server with timeouts
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      nil, // Use default ServeMux
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		logger.Fatalf("Failed to start server: %v", err)
	}
}

type webhookHandler struct {
	logger      *clog.Logger
	queueClient workqueue.Client
	cfg         Config
}

func (h *webhookHandler) handleWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := h.logger.With(
		"method", r.Method,
		"path", r.URL.Path,
		"remote", r.RemoteAddr,
	)

	logger.Debug("Received webhook request")

	if r.Method != http.MethodPost {
		logger.With("method", r.Method).Error("Invalid HTTP method for webhook")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get the event type
	eventType := github.WebHookType(r)
	if eventType == "" {
		logger.Error("Missing X-GitHub-Event header")
		http.Error(w, "Missing event type", http.StatusBadRequest)
		return
	}

	logger.With("event_type", eventType).Info("Processing GitHub event")

	// Validate payload and parse the webhook event
	payload, err := github.ValidatePayload(r, []byte(h.cfg.WebhookSecret))
	if err != nil {
		logger.Errorf("Failed to validate webhook signature: %v", err)
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	// Parse the webhook event
	event, err := github.ParseWebHook(eventType, payload)
	if err != nil {
		logger.Errorf("Failed to parse webhook: %v", err)
		http.Error(w, "Failed to parse webhook", http.StatusBadRequest)
		return
	}

	// Extract resource URL
	resourceURL := h.extractResourceURL(event, logger)
	if resourceURL == "" {
		// Event filtered or not supported
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			logger.Errorf("Failed to write response: %v", err)
		}
		return
	}

	// Note: Rate limiting could be added here if needed
	// For now, we rely on GitHub's webhook delivery rate limiting

	// Queue the work item
	_, err = h.queueClient.Process(ctx, &workqueue.ProcessRequest{
		Key:      resourceURL,
		Priority: 0, // Default priority
	})
	if err != nil {
		logger.With("key", resourceURL).Errorf("Failed to queue work item: %v", err)
		http.Error(w, "Failed to queue work item", http.StatusInternalServerError)
		return
	}

	logger.With("key", resourceURL).Info("Successfully queued work item from webhook")

	// Always return 200 OK for webhooks
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("OK")); err != nil {
		logger.Errorf("Failed to write response: %v", err)
	}
}

// extractResourceURL extracts the resource URL from various GitHub webhook events
func (h *webhookHandler) extractResourceURL(event interface{}, logger *clog.Logger) string {
	switch e := event.(type) {
	case *github.IssuesEvent:
		if h.cfg.ResourceFilter != "" && h.cfg.ResourceFilter != "issues" {
			logger.With("resource_filter", h.cfg.ResourceFilter).Debug("Ignoring issues event due to resource filter")
			return ""
		}
		return fmt.Sprintf("https://github.com/%s/%s/issues/%d",
			e.GetRepo().GetOwner().GetLogin(),
			e.GetRepo().GetName(),
			e.GetIssue().GetNumber())

	case *github.IssueCommentEvent:
		isPR := e.GetIssue().IsPullRequest()
		if h.cfg.ResourceFilter != "" {
			if (isPR && h.cfg.ResourceFilter != "pull_requests") || (!isPR && h.cfg.ResourceFilter != "issues") {
				logger.With("resource_filter", h.cfg.ResourceFilter, "is_pr", isPR).Debug("Ignoring comment event due to resource filter")
				return ""
			}
		}
		resourceType := "issues"
		if isPR {
			resourceType = "pull"
		}
		return fmt.Sprintf("https://github.com/%s/%s/%s/%d",
			e.GetRepo().GetOwner().GetLogin(),
			e.GetRepo().GetName(),
			resourceType,
			e.GetIssue().GetNumber())

	case *github.PullRequestEvent:
		if h.cfg.ResourceFilter != "" && h.cfg.ResourceFilter != "pull_requests" {
			logger.With("resource_filter", h.cfg.ResourceFilter).Debug("Ignoring pull request event due to resource filter")
			return ""
		}
		return fmt.Sprintf("https://github.com/%s/%s/pull/%d",
			e.GetRepo().GetOwner().GetLogin(),
			e.GetRepo().GetName(),
			e.GetPullRequest().GetNumber())

	case *github.PullRequestReviewEvent:
		if h.cfg.ResourceFilter != "" && h.cfg.ResourceFilter != "pull_requests" {
			logger.With("resource_filter", h.cfg.ResourceFilter).Debug("Ignoring pull request review event due to resource filter")
			return ""
		}
		return fmt.Sprintf("https://github.com/%s/%s/pull/%d",
			e.GetRepo().GetOwner().GetLogin(),
			e.GetRepo().GetName(),
			e.GetPullRequest().GetNumber())

	case *github.PullRequestReviewCommentEvent:
		if h.cfg.ResourceFilter != "" && h.cfg.ResourceFilter != "pull_requests" {
			logger.With("resource_filter", h.cfg.ResourceFilter).Debug("Ignoring pull request review comment event due to resource filter")
			return ""
		}
		return fmt.Sprintf("https://github.com/%s/%s/pull/%d",
			e.GetRepo().GetOwner().GetLogin(),
			e.GetRepo().GetName(),
			e.GetPullRequest().GetNumber())

	case *github.CheckRunEvent:
		if h.cfg.ResourceFilter != "" && h.cfg.ResourceFilter != "pull_requests" {
			logger.With("resource_filter", h.cfg.ResourceFilter).Debug("Ignoring check run event due to resource filter")
			return ""
		}
		if len(e.GetCheckRun().PullRequests) > 0 {
			return fmt.Sprintf("https://github.com/%s/%s/pull/%d",
				e.GetRepo().GetOwner().GetLogin(),
				e.GetRepo().GetName(),
				e.GetCheckRun().PullRequests[0].GetNumber())
		}
		return ""

	case *github.CheckSuiteEvent:
		if h.cfg.ResourceFilter != "" && h.cfg.ResourceFilter != "pull_requests" {
			logger.With("resource_filter", h.cfg.ResourceFilter).Debug("Ignoring check suite event due to resource filter")
			return ""
		}
		if len(e.GetCheckSuite().PullRequests) > 0 {
			return fmt.Sprintf("https://github.com/%s/%s/pull/%d",
				e.GetRepo().GetOwner().GetLogin(),
				e.GetRepo().GetName(),
				e.GetCheckSuite().PullRequests[0].GetNumber())
		}
		return ""

	case *github.PingEvent:
		logger.With("hook_id", e.GetHookID()).Info("Received GitHub ping event")
		return ""

	default:
		logger.With("event_type", fmt.Sprintf("%T", event)).Debug("Ignoring unhandled event type")
		return ""
	}
}
