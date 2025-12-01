/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package githubreconciler

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/go-github/v75/github"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/chainguard-dev/clog"
	"github.com/chainguard-dev/terraform-infra-common/pkg/workqueue"
)

// ReconcilerFunc is the function signature for GitHub resource reconcilers.
// It receives the parsed resource information and appropriate GitHub client,
// and returns an error if reconciliation fails.
type ReconcilerFunc func(ctx context.Context, res *Resource, gh *github.Client) error

// Resource represents a parsed GitHub resource (issue, pull request, or path).
type Resource struct {
	// Owner is the GitHub organization or user.
	Owner string

	// Repo is the repository name.
	Repo string

	// Number is the issue or pull request number.
	// Only set for ResourceTypeIssue and ResourceTypePullRequest.
	Number int

	// Type indicates the resource type.
	Type ResourceType

	// URL is the original URL that was parsed.
	URL string

	// Ref is the branch, tag, or commit SHA.
	// Only set for ResourceTypePath.
	Ref string

	// Path is the file or directory path.
	// Only set for ResourceTypePath.
	Path string
}

// ResourceType represents the type of GitHub resource.
type ResourceType string

const (
	// ResourceTypeIssue represents a GitHub issue.
	ResourceTypeIssue ResourceType = "issue"

	// ResourceTypePullRequest represents a GitHub pull request.
	ResourceTypePullRequest ResourceType = "pull_request"

	// ResourceTypePath represents a file or directory path in a repository.
	ResourceTypePath ResourceType = "path"

	// grpcRateLimitRetryDuration is the base duration to wait before retrying
	// when a gRPC ResourceExhausted error is encountered.
	grpcRateLimitRetryDuration = 2 * time.Minute
)

// String returns the string representation of the resource.
func (r *Resource) String() string {
	switch r.Type {
	case ResourceTypeIssue, ResourceTypePullRequest:
		return fmt.Sprintf("%s/%s#%d", r.Owner, r.Repo, r.Number)
	case ResourceTypePath:
		return fmt.Sprintf("%s/%s@%s:%s", r.Owner, r.Repo, r.Ref, r.Path)
	default:
		return fmt.Sprintf("%s/%s", r.Owner, r.Repo)
	}
}

// addJitter adds random jitter to a duration to avoid thundering herd.
// Jitter is 0% to +10% of the base duration.
//
//nolint:gosec // Using weak random for jitter is fine, not cryptographic
func addJitter(d time.Duration) time.Duration {
	// Add jitter between 0% and +10%
	jitter := time.Duration(rand.Int63n(int64(d / 10)))
	return d + jitter
}

// Reconciler manages the reconciliation of GitHub resources.
type Reconciler struct {
	workqueue.UnimplementedWorkqueueServiceServer

	// reconcileFunc is the reconciler for all resource types.
	reconcileFunc ReconcilerFunc

	// clientCache manages GitHub API clients per repository.
	clientCache *ClientCache

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

	return r
}

// Reconcile processes the given resource URL.
func (r *Reconciler) Reconcile(ctx context.Context, url string) error {
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
			delay := addJitter(time.Until(resetTime))
			clog.FromContext(ctx).With("reset_at", resetTime).
				Warn("Rate limited, requeueing after rate limit reset")
			return workqueue.RequeueAfter(delay)
		}

		// Check if it's an abuse rate limit error
		var abuseRateLimitErr *github.AbuseRateLimitError
		if errors.As(err, &abuseRateLimitErr) {
			// GitHub wants us to slow down - use retry after if provided, otherwise use a conservative 1 minute
			retryAfter := time.Minute
			if abuseRateLimitErr.RetryAfter != nil {
				retryAfter = *abuseRateLimitErr.RetryAfter
			}
			delay := addJitter(retryAfter)
			clog.FromContext(ctx).With("retry_after", delay).
				Warn("Abuse rate limit detected, requeueing after retry period")
			return workqueue.RequeueAfter(delay)
		}

		// Check if it's a gRPC ResourceExhausted error
		if status.Code(err) == codes.ResourceExhausted {
			// Resource exhausted - use a conservative retry delay
			delay := addJitter(grpcRateLimitRetryDuration)
			clog.FromContext(ctx).With("retry_after", delay).
				Warn("gRPC ResourceExhausted detected, requeueing after retry period")
			return workqueue.RequeueAfter(delay)
		}
	}
	return err
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
