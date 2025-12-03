/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"os/signal"
	"regexp"
	"syscall"
	"time"

	"github.com/chainguard-dev/clog"
	_ "github.com/chainguard-dev/clog/gcp/init"
	"github.com/chainguard-dev/terraform-infra-common/modules/github-path-reconciler/internal/patterns"
	"github.com/chainguard-dev/terraform-infra-common/pkg/githubreconciler"
	"github.com/chainguard-dev/terraform-infra-common/pkg/httpmetrics"
	"github.com/chainguard-dev/terraform-infra-common/pkg/workqueue"
	"github.com/sethvargo/go-envconfig"
	"golang.org/x/oauth2"
	"golang.org/x/sync/errgroup"
)

type config struct {
	// GitHub configuration
	GitHubOwner string `env:"GITHUB_OWNER,required"`
	GitHubRepo  string `env:"GITHUB_REPO,required"`

	// Octo STS configuration
	OctoSTSIdentity string `env:"OCTO_STS_IDENTITY,required"`

	// Workqueue configuration
	WorkqueueAddr string `env:"WORKQUEUE_ADDR,required"`

	// Path patterns (JSON array)
	PathPatterns string `env:"PATH_PATTERNS,required"`

	// Period in minutes for time bucketing
	PeriodMinutes int `env:"PERIOD_MINUTES,required"`
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	go httpmetrics.ServeMetrics()
	defer httpmetrics.SetupTracer(ctx)()
	httpmetrics.SetBuckets(map[string]string{
		"api.github.com": "github",
		"octo-sts.dev":   "octosts",
	})

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

	handler := &cronHandler{
		clientCache: githubreconciler.NewClientCache(func(ctx context.Context, org, repo string) (oauth2.TokenSource, error) {
			return githubreconciler.NewRepoTokenSource(ctx, cfg.OctoSTSIdentity, org, repo), nil
		}),
		wqClient:      wqClient,
		owner:         cfg.GitHubOwner,
		repo:          cfg.GitHubRepo,
		patterns:      pats,
		periodMinutes: cfg.PeriodMinutes,
	}

	clog.InfoContextf(ctx, "Starting cron run for %s/%s", cfg.GitHubOwner, cfg.GitHubRepo)
	if err := handler.run(ctx); err != nil {
		clog.FatalContextf(ctx, "Cron run failed: %v", err)
	}
	clog.InfoContextf(ctx, "Cron run complete")
}

type cronHandler struct {
	clientCache   *githubreconciler.ClientCache
	wqClient      workqueue.Client
	owner         string
	repo          string
	patterns      []*regexp.Regexp
	periodMinutes int
}

func (h *cronHandler) run(ctx context.Context) error {
	runTimestamp := time.Now().Unix()

	// Get GitHub client from cache
	ghClient, err := h.clientCache.Get(ctx, h.owner, h.repo)
	if err != nil {
		return fmt.Errorf("failed to get GitHub client: %w", err)
	}

	// Get the repository to determine the default branch
	repo, _, err := ghClient.Repositories.Get(ctx, h.owner, h.repo)
	if err != nil {
		return fmt.Errorf("failed to get repository: %w", err)
	}
	defaultBranch := repo.GetDefaultBranch()

	// Get repository tree at default branch
	tree, _, err := ghClient.Git.GetTree(ctx, h.owner, h.repo, defaultBranch, true)
	if err != nil {
		return fmt.Errorf("failed to get repository tree: %w", err)
	}

	// Accumulate unique keys
	keySet := make(map[string]struct{})

	// Process each file in the tree
	for _, entry := range tree.Entries {
		if entry.GetType() != "blob" {
			continue // Skip non-files
		}

		path := entry.GetPath()

		// Try to match against each pattern
		for _, regex := range h.patterns {
			matches := regex.FindStringSubmatch(path)
			if len(matches) <= 1 {
				continue
			}

			capturedPath := matches[1] // First capture group

			// Build resource URL using the default branch
			url := fmt.Sprintf("https://github.com/%s/%s/blob/%s/%s", h.owner, h.repo, defaultBranch, capturedPath)

			// Add to key set (deduplicates automatically)
			keySet[url] = struct{}{}
			break // Only match first pattern
		}
	}

	// Enqueue all unique keys with their computed delays
	eg, egCtx := errgroup.WithContext(ctx)

	for url := range keySet {
		url := url // capture for goroutine
		eg.Go(func() error {
			// Compute delay bucket
			delay := h.computeDelay(url, runTimestamp)

			// Process the key via workqueue
			_, err := h.wqClient.Process(egCtx, &workqueue.ProcessRequest{
				Key:          url,
				Priority:     0,
				DelaySeconds: int64(delay.Seconds()),
			})
			if err != nil {
				clog.ErrorContextf(egCtx, "Failed to process key %q: %v", url, err)
				return err
			}

			clog.InfoContextf(egCtx, "Enqueued %q with delay %v", url, delay)
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return fmt.Errorf("failed to enqueue all keys: %w", err)
	}

	return nil
}

func (h *cronHandler) computeDelay(key string, runTimestamp int64) time.Duration {
	// Hash the key + timestamp to get a consistent bucket.
	// We include runTimestamp in the hash so that we don't end up with the same
	// key ordering every single time things run.
	hashInput := fmt.Sprintf("%s-%d", key, runTimestamp)
	hash := sha256.Sum256([]byte(hashInput))
	hashValue := binary.BigEndian.Uint64(hash[:8])

	// Compute bucket (periodMinutes is validated to be 60-1440, so conversion is safe)
	bucket := int(hashValue % uint64(h.periodMinutes)) //nolint:gosec // G115: periodMinutes validated range

	return time.Duration(bucket) * time.Minute
}
