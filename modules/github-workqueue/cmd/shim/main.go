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

	// GitHub webhook handler
	http.HandleFunc("/webhook", webhookHandler(logger, queueClient, cfg))

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

func webhookHandler(logger *clog.Logger, queueClient workqueue.Client, cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.With(
			"method", r.Method,
			"path", r.URL.Path,
			"remote", r.RemoteAddr,
		).Debug("Received webhook request")

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
		payload, err := github.ValidatePayload(r, []byte(cfg.WebhookSecret))
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

		// Extract resource URL from event
		key := extractResourceURL(event, cfg.ResourceFilter, logger)

		// Queue the work item if we have a key
		if key != "" {
			_, err := queueClient.Process(r.Context(), &workqueue.ProcessRequest{
				Key:      key,
				Priority: 0, // Default priority
			})
			if err != nil {
				logger.With("key", key).Errorf("Failed to queue work item: %v", err)
				http.Error(w, "Failed to queue work item", http.StatusInternalServerError)
				return
			}

			logger.With("key", key).Info("Successfully queued work item from webhook")
		}

		// Always return 200 OK for webhooks
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			logger.Errorf("Failed to write response: %v", err)
		}
	}
}

// extractResourceURL extracts the resource URL from various GitHub webhook events
func extractResourceURL(event interface{}, resourceFilter string, logger *clog.Logger) string {
	switch e := event.(type) {
	case *github.IssuesEvent:
		if resourceFilter != "" && resourceFilter != "issues" {
			logger.With("resource_filter", resourceFilter).Debug("Ignoring issues event due to resource filter")
			return ""
		}
		return fmt.Sprintf("https://github.com/%s/%s/issues/%d",
			e.GetRepo().GetOwner().GetLogin(),
			e.GetRepo().GetName(),
			e.GetIssue().GetNumber())

	case *github.IssueCommentEvent:
		isPR := e.GetIssue().IsPullRequest()
		if resourceFilter != "" {
			if (isPR && resourceFilter != "pull_requests") || (!isPR && resourceFilter != "issues") {
				logger.With("resource_filter", resourceFilter, "is_pr", isPR).Debug("Ignoring comment event due to resource filter")
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
		if resourceFilter != "" && resourceFilter != "pull_requests" {
			logger.With("resource_filter", resourceFilter).Debug("Ignoring pull request event due to resource filter")
			return ""
		}
		return fmt.Sprintf("https://github.com/%s/%s/pull/%d",
			e.GetRepo().GetOwner().GetLogin(),
			e.GetRepo().GetName(),
			e.GetPullRequest().GetNumber())

	case *github.PullRequestReviewEvent:
		if resourceFilter != "" && resourceFilter != "pull_requests" {
			logger.With("resource_filter", resourceFilter).Debug("Ignoring pull request review event due to resource filter")
			return ""
		}
		return fmt.Sprintf("https://github.com/%s/%s/pull/%d",
			e.GetRepo().GetOwner().GetLogin(),
			e.GetRepo().GetName(),
			e.GetPullRequest().GetNumber())

	case *github.PullRequestReviewCommentEvent:
		if resourceFilter != "" && resourceFilter != "pull_requests" {
			logger.With("resource_filter", resourceFilter).Debug("Ignoring pull request review comment event due to resource filter")
			return ""
		}
		return fmt.Sprintf("https://github.com/%s/%s/pull/%d",
			e.GetRepo().GetOwner().GetLogin(),
			e.GetRepo().GetName(),
			e.GetPullRequest().GetNumber())

	case *github.CheckRunEvent:
		if resourceFilter != "" && resourceFilter != "pull_requests" {
			logger.With("resource_filter", resourceFilter).Debug("Ignoring check run event due to resource filter")
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
		if resourceFilter != "" && resourceFilter != "pull_requests" {
			logger.With("resource_filter", resourceFilter).Debug("Ignoring check suite event due to resource filter")
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
