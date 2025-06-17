/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package githubreconciler

import (
	"context"
	"fmt"

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
	// issueFunc is the reconciler for issues.
	issueFunc ReconcilerFunc

	// prFunc is the reconciler for pull requests.
	prFunc ReconcilerFunc

	// clientCache manages GitHub API clients per repository.
	clientCache *ClientCache

	// stateManager handles state persistence in GitHub comments.
	stateManager *StateManager
}

// Option configures a Reconciler.
type Option func(*Reconciler)

// WithIssueReconciler sets the reconciler function for issues.
func WithIssueReconciler(f ReconcilerFunc) Option {
	return func(r *Reconciler) {
		r.issueFunc = f
	}
}

// WithPullRequestReconciler sets the reconciler function for pull requests.
func WithPullRequestReconciler(f ReconcilerFunc) Option {
	return func(r *Reconciler) {
		r.prFunc = f
	}
}

// WithStateManager sets a custom state manager.
func WithStateManager(sm *StateManager) Option {
	return func(r *Reconciler) {
		r.stateManager = sm
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
	// Parse the URL to extract resource information
	resource, err := ParseURL(url)
	if err != nil {
		return fmt.Errorf("parsing URL: %w", err)
	}

	// Get the appropriate GitHub client
	client, err := r.clientCache.Get(ctx, resource.Owner, resource.Repo)
	if err != nil {
		return fmt.Errorf("getting GitHub client: %w", err)
	}

	// Route to the appropriate reconciler
	var reconcileFunc ReconcilerFunc
	switch resource.Type {
	case ResourceTypeIssue:
		reconcileFunc = r.issueFunc
	case ResourceTypePullRequest:
		reconcileFunc = r.prFunc
	default:
		return fmt.Errorf("unknown resource type: %s", resource.Type)
	}

	if reconcileFunc == nil {
		return fmt.Errorf("no reconciler configured for %s", resource.Type)
	}

	// Execute the reconciliation
	return reconcileFunc(ctx, resource, client)
}

// GetStateManager returns the state manager for accessing state operations.
func (r *Reconciler) GetStateManager() *StateManager {
	return r.stateManager
}
