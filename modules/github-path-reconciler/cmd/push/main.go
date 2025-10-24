/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"fmt"
	"os/signal"
	"regexp"
	"strings"
	"syscall"

	"github.com/chainguard-dev/clog"
	_ "github.com/chainguard-dev/clog/gcp/init"
	"github.com/chainguard-dev/terraform-infra-common/modules/github-path-reconciler/internal/patterns"
	"github.com/chainguard-dev/terraform-infra-common/pkg/githubreconciler"
	"github.com/chainguard-dev/terraform-infra-common/pkg/httpmetrics"
	"github.com/chainguard-dev/terraform-infra-common/pkg/workqueue"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/go-github/v75/github"
	"github.com/sethvargo/go-envconfig"
	"golang.org/x/oauth2"
	"golang.org/x/sync/errgroup"
)

type config struct {
	Port int `env:"PORT,default=8080"`

	// Workqueue configuration
	WorkqueueAddr string `env:"WORKQUEUE_ADDR,required"`

	// Path patterns (JSON array)
	PathPatterns string `env:"PATH_PATTERNS,required"`

	// Octo STS identity
	OctoIdentity string `env:"OCTO_IDENTITY,required"`
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	go httpmetrics.ServeMetrics()
	defer httpmetrics.SetupTracer(ctx)()

	var cfg config
	if err := envconfig.Process(ctx, &cfg); err != nil {
		clog.FatalContextf(ctx, "Failed to process environment: %v", err)
	}

	// Parse path patterns
	pats, err := patterns.Parse(cfg.PathPatterns)
	if err != nil {
		clog.FatalContextf(ctx, "Failed to parse path patterns: %v", err)
	}

	// Set up workqueue client
	wqClient, err := workqueue.NewWorkqueueClient(ctx, cfg.WorkqueueAddr)
	if err != nil {
		clog.FatalContextf(ctx, "Failed to create workqueue client: %v", err)
	}
	defer wqClient.Close()

	clientCache := githubreconciler.NewClientCache(func(ctx context.Context, org, repo string) (oauth2.TokenSource, error) {
		return githubreconciler.NewRepoTokenSource(ctx, cfg.OctoIdentity, org, repo), nil
	})

	handler := &pushHandler{
		clientCache: clientCache,
		wqClient:    wqClient,
		patterns:    pats,
	}

	// Set up Cloud Events receiver
	ceClient, err := cloudevents.NewClientHTTP(cloudevents.WithPort(cfg.Port))
	if err != nil {
		clog.FatalContextf(ctx, "Failed to create CloudEvents client: %v", err)
	}

	clog.InfoContextf(ctx, "Starting push listener on port %d", cfg.Port)
	if err := ceClient.StartReceiver(ctx, handler.handlePushEvent); err != nil {
		clog.FatalContextf(ctx, "Failed to start receiver: %v", err)
	}
}

type pushHandler struct {
	clientCache *githubreconciler.ClientCache
	wqClient    workqueue.Client
	patterns    []*regexp.Regexp
}

func (h *pushHandler) handlePushEvent(ctx context.Context, event cloudevents.Event) error {
	log := clog.FromContext(ctx)

	// Log all events we receive for debugging
	log.Infof("Received event: type=%s, source=%s, subject=%s", event.Type(), event.Source(), event.Subject())

	// Filter for push events in code
	if event.Type() != "dev.chainguard.github.push" {
		log.Infof("Ignoring non-push event: %s", event.Type())
		return nil
	}

	// Unwrap the event envelope - the trampoline wraps the GitHub payload
	var envelope struct {
		Body github.PushEvent `json:"body"`
	}
	if err := event.DataAs(&envelope); err != nil {
		return fmt.Errorf("failed to parse event envelope: %w", err)
	}

	// Use the push event from the envelope body
	pushEvent := envelope.Body

	owner := pushEvent.GetRepo().GetOwner().GetLogin()
	repo := pushEvent.GetRepo().GetName()
	ref := pushEvent.GetRef()
	before := pushEvent.GetBefore()
	after := pushEvent.GetAfter()
	defaultBranch := pushEvent.GetRepo().GetDefaultBranch()

	log = log.With(
		"owner", owner,
		"repo", repo,
		"ref", ref,
		"before", before,
		"after", after,
		"default_branch", defaultBranch,
	)
	ctx = clog.WithLogger(ctx, log)

	// Extract branch name from ref (refs/heads/main -> main)
	branch := strings.TrimPrefix(ref, "refs/heads/")

	// Only process pushes to the default branch
	if branch != defaultBranch {
		log.Infof("Ignoring push to non-default branch %q (default is %q)", branch, defaultBranch)
		return nil
	}

	log.Infof("Processing push event for %s/%s on default branch %q", owner, repo, defaultBranch)

	// Get GitHub client
	ghClient, err := h.clientCache.Get(ctx, owner, repo)
	if err != nil {
		return fmt.Errorf("failed to get GitHub client: %w", err)
	}

	// Use the GitHub API to compare commits to get all changed files
	// This handles all merge strategies correctly (merge commits, squash, rebase)
	comparison, _, err := ghClient.Repositories.CompareCommits(ctx, owner, repo, before, after, &github.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to compare commits: %w", err)
	}

	// Collect all changed files from the comparison
	changedFiles := make(map[string]struct{})
	for _, file := range comparison.Files {
		changedFiles[file.GetFilename()] = struct{}{}
	}

	log.Infof("Processing %d changed files", len(changedFiles))

	// Extract keys from changed files
	keySet := make(map[string]struct{})
	for file := range changedFiles {
		for _, regex := range h.patterns {
			matches := regex.FindStringSubmatch(file)
			if len(matches) <= 1 {
				continue
			}

			capturedPath := matches[1] // First capture group

			// Build resource URL using the default branch
			url := fmt.Sprintf("https://github.com/%s/%s/blob/%s/%s", owner, repo, defaultBranch, capturedPath)

			// Add to key set (deduplicates automatically)
			keySet[url] = struct{}{}
			break // Only match first pattern
		}
	}

	log.Infof("Enqueueing %d unique keys", len(keySet))

	// Enqueue all unique keys
	eg, egCtx := errgroup.WithContext(ctx)

	for url := range keySet {
		url := url // capture for goroutine
		eg.Go(func() error {
			_, err := h.wqClient.Process(egCtx, &workqueue.ProcessRequest{
				Key:      url,
				Priority: 100, // Process push events immediately
			})
			if err != nil {
				clog.ErrorContextf(egCtx, "Failed to process key %q: %v", url, err)
				return err
			}

			clog.InfoContextf(egCtx, "Enqueued %q", url)
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return fmt.Errorf("failed to enqueue all keys: %w", err)
	}

	return nil
}
