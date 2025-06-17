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

// IssueState represents the state we track for issues
type IssueState struct {
	IssueState   string `json:"issue_state"`
	CommentCount int    `json:"comment_count"`
}

// NewIssueReconciler creates a sample issue reconciler that posts status reports
func NewIssueReconciler(stateManager *githubreconciler.StateManager) githubreconciler.ReconcilerFunc {
	return func(ctx context.Context, res *githubreconciler.Resource, gh *github.Client) error {
		log := clog.FromContext(ctx).With("resource", res.String())

		// Create state for this resource
		state := githubreconciler.NewState[IssueState](stateManager.Identity(), gh, res)

		// Fetch current state
		currentState, err := state.Fetch(ctx)
		if err != nil {
			log.Errorf("Failed to fetch current state: %v", err)
			// Continue with fresh state
		}

		// Initialize new state
		newState := &IssueState{}

		// Fetch the issue
		issue, _, err := gh.Issues.Get(ctx, res.Owner, res.Repo, res.Number)
		if err != nil {
			return fmt.Errorf("fetching issue: %w", err)
		}

		newState.IssueState = issue.GetState()

		// Log issue details
		log.With(
			"title", issue.GetTitle(),
			"state", issue.GetState(),
			"author", issue.GetUser().GetLogin(),
		).Info("Processing GitHub issue")

		// Count comments
		opts := &github.IssueListCommentsOptions{
			ListOptions: github.ListOptions{
				PerPage: 100,
			},
		}

		commentCount := 0
		for {
			comments, resp, err := gh.Issues.ListComments(ctx, res.Owner, res.Repo, res.Number, opts)
			if err != nil {
				return fmt.Errorf("listing comments: %w", err)
			}

			commentCount += len(comments)

			if resp.NextPage == 0 {
				break
			}
			opts.Page = resp.NextPage
		}

		newState.CommentCount = commentCount

		// Check if state has changed
		if currentState != nil &&
			currentState.IssueState == newState.IssueState &&
			currentState.CommentCount == newState.CommentCount {
			log.Info("Issue state unchanged, skipping update")
			return nil
		}

		// Build a status message
		message := fmt.Sprintf(`## Issue Status Report

**Issue #%d**: %s
**State**: %s
**Comments**: %d

This issue has been analyzed by the GitHub reconciler.`,
			issue.GetNumber(),
			issue.GetTitle(),
			issue.GetState(),
			commentCount)

		// Commit the new state
		if err := state.Commit(ctx, newState, message); err != nil {
			return fmt.Errorf("committing state: %w", err)
		}

		log.Info("Successfully reconciled issue")
		return nil
	}
}
