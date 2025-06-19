/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package examples

import (
	"context"
	"fmt"

	"github.com/chainguard-dev/clog"
	"github.com/chainguard-dev/terraform-infra-common/pkg/githubreconciler"
	"github.com/google/go-github/v72/github"
)

// NewCombinedReconciler creates a reconciler that handles both issues and pull requests
func NewCombinedReconciler(stateManager *githubreconciler.StateManager) githubreconciler.ReconcilerFunc {
	// Create separate reconcilers for issues and PRs
	issueReconciler := NewIssueReconciler(stateManager)
	prReconciler := NewPullRequestReconciler(stateManager)

	// Return a single reconciler that routes based on resource type
	return func(ctx context.Context, res *githubreconciler.Resource, gh *github.Client) error {
		log := clog.FromContext(ctx).With("resource", res.String(), "type", res.Type)

		switch res.Type {
		case githubreconciler.ResourceTypeIssue:
			log.Info("Reconciling issue")
			return issueReconciler(ctx, res, gh)

		case githubreconciler.ResourceTypePullRequest:
			log.Info("Reconciling pull request")
			return prReconciler(ctx, res, gh)

		default:
			return fmt.Errorf("unsupported resource type: %s", res.Type)
		}
	}
}