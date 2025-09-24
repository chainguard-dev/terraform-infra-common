/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package githubreconciler

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/chainguard-dev/clog"
	"github.com/chainguard-dev/terraform-infra-common/pkg/workqueue"
	"github.com/google/go-github/v72/github"
)

// ReconcilerFunc is the function signature for GitHub resource reconcilers.
// It receives the parsed resource information and appropriate GitHub client,
// and returns an error if reconciliation fails.
type ReconcilerFunc func(ctx context.Context, res *Resource, gh *github.Client) error

// Resource represents a parsed GitHub resource (issue or pull request).
type Resource struct {
	// Owner is the GitHub organization or user.
	Owner string

	// Repo is the repository name.
	Repo string

	// Number is the issue or pull request number.
	Number int

	// Type indicates whether this is an issue or pull request.
	Type ResourceType

	// URL is the original URL that was parsed.
	URL string
}

// ResourceType represents the type of GitHub resource.
type ResourceType string

const (
	// ResourceTypeIssue represents a GitHub issue.
	ResourceTypeIssue ResourceType = "issue"

	// ResourceTypePullRequest represents a GitHub pull request.
	ResourceTypePullRequest ResourceType = "pull_request"
)

// String returns the string representation of the resource.
func (r *Resource) String() string {
	return fmt.Sprintf("%s/%s#%d", r.Owner, r.Repo, r.Number)
}

// Reconciler manages the reconciliation of GitHub resources.
type Reconciler struct {
	workqueue.UnimplementedWorkqueueServiceServer

	// reconcileFunc is the reconciler for all resource types.
	reconcileFunc ReconcilerFunc

	// clientCache manages GitHub API clients per repository.
	clientCache *ClientCache

	// stateManager handles state persistence in GitHub comments.
	stateManager *StateManager

	// useOrgScopedCredentials indicates whether to use org-scoped instead of repo-scoped credentials.
	useOrgScopedCredentials bool
}

// Option configures a Reconciler.
type Option func(*Reconciler)

// WithReconciler sets the reconciler function for all resource types.
func WithReconciler(f ReconcilerFunc) Option {
	return func(r *Reconciler) {
		r.reconcileFunc = f
	}
}

// WithStateManager sets a custom state manager.
func WithStateManager(sm *StateManager) Option {
	return func(r *Reconciler) {
		r.stateManager = sm
	}
}

// WithOrgScopedCredentials configures the reconciler to use org-scoped credentials
// instead of repo-scoped credentials. When enabled, the same GitHub client will be
// used for all repositories within an organization.
func WithOrgScopedCredentials() Option {
	return func(r *Reconciler) {
		r.useOrgScopedCredentials = true
	}
}

// NewReconciler creates a new Reconciler with the given options.
func NewReconciler(cc *ClientCache, opts ...Option) *Reconciler {
	r := &Reconciler{
		clientCache: cc,
	}

	for _, opt := range opts {
		opt(r)
	}

	// Use a default state manager if none provided
	if r.stateManager == nil {
		r.stateManager = NewStateManager("github-reconciler")
	}

	return r
}

// Reconcile processes the given resource URL.
func (r *Reconciler) Reconcile(ctx context.Context, url string) error {
	// Add the key to the logger context for filtering
	ctx = clog.WithLogger(ctx, clog.FromContext(ctx).With("key", url))

	// Parse the URL to extract resource information
	resource, err := ParseURL(url)
	if err != nil {
		return fmt.Errorf("parsing URL: %w", err)
	}

	// Get the appropriate GitHub client
	// When using org-scoped credentials, pass .github as repo for Octo STS
	repo := resource.Repo
	if r.useOrgScopedCredentials {
		// Drop the repo component to make it org-scoped.
		repo = ""
	}
	client, err := r.clientCache.Get(ctx, resource.Owner, repo)
	if err != nil {
		return fmt.Errorf("getting GitHub client: %w", err)
	}

	if r.reconcileFunc == nil {
		return fmt.Errorf("no reconciler configured")
	}

	// Execute the reconciliation
	err = r.reconcileFunc(ctx, resource, client)
	if err != nil {
		// Check if it's a rate limit error from any GitHub API call
		var rateLimitErr *github.RateLimitError
		if errors.As(err, &rateLimitErr) {
			// Calculate duration until rate limit resets
			resetTime := rateLimitErr.Rate.Reset.Time
			clog.FromContext(ctx).With("reset_at", resetTime).
				Warn("Rate limited, requeueing after rate limit reset")
			return workqueue.RequeueAfter(time.Until(resetTime))
		}

		// Check if it's an abuse rate limit error
		var abuseRateLimitErr *github.AbuseRateLimitError
		if errors.As(err, &abuseRateLimitErr) {
			// GitHub wants us to slow down - use retry after if provided, otherwise use a conservative 1 minute
			retryAfter := time.Minute
			if abuseRateLimitErr.RetryAfter != nil {
				retryAfter = *abuseRateLimitErr.RetryAfter
			}
			clog.FromContext(ctx).With("retry_after", retryAfter).
				Warn("Abuse rate limit detected, requeueing after retry period")
			return workqueue.RequeueAfter(retryAfter)
		}
	}
	return err
}

// GetStateManager returns the state manager for accessing state operations.
func (r *Reconciler) GetStateManager() *StateManager {
	return r.stateManager
}

// Process implements the WorkqueueService.Process RPC.
func (r *Reconciler) Process(ctx context.Context, req *workqueue.ProcessRequest) (*workqueue.ProcessResponse, error) {
	clog.InfoContextf(ctx, "Processing GitHub resource: %s (priority: %d)", req.Key, req.Priority)

	// Call the reconciler
	err := r.Reconcile(ctx, req.Key)
	if err != nil {
		// Check if we can extract a requeue delay from the error
		if delay, ok := workqueue.GetRequeueDelay(err); ok {
			clog.InfoContextf(ctx, "Reconciliation requested requeue after %v for key: %s", delay, req.Key)
			return &workqueue.ProcessResponse{
				RequeueAfterSeconds: int64(delay.Seconds()),
			}, nil
		}

		// Check if this is a non-retriable error
		if details := workqueue.GetNonRetriableDetails(err); details != nil {
			clog.WarnContextf(ctx, "Reconciliation failed with non-retriable error for key %s: %v (reason: %s)", req.Key, err, details.Message)
			// Return nil error to indicate successful processing (but don't retry)
			return &workqueue.ProcessResponse{}, nil
		}

		// Regular error - will be retried with exponential backoff
		clog.ErrorContextf(ctx, "Reconciliation failed for key %s: %v", req.Key, err)
		return nil, err
	}

	clog.InfoContextf(ctx, "Successfully reconciled GitHub resource: %s", req.Key)
	return &workqueue.ProcessResponse{}, nil
}
