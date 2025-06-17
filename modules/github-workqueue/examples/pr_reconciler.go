/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package examples

import (
	"context"
	"fmt"
	"strings"

	"github.com/chainguard-dev/clog"
	"github.com/chainguard-dev/terraform-infra-common/pkg/githubreconciler"
	"github.com/google/go-github/v72/github"
)

// PullRequestState represents the state we track for pull requests
type PullRequestState struct {
	PRState      string            `json:"pr_state"`
	Mergeable    *bool             `json:"mergeable"`
	CommitCount  int               `json:"commit_count"`
	CommentCount int               `json:"comment_count"`
	ReviewCount  int               `json:"review_count"`
	FileCount    int               `json:"file_count"`
	ChecksStatus map[string]string `json:"checks_status"`
}

// NewPullRequestReconciler creates a sample PR reconciler that posts comprehensive status reports
func NewPullRequestReconciler(stateManager *githubreconciler.StateManager) githubreconciler.ReconcilerFunc {
	return func(ctx context.Context, res *githubreconciler.Resource, gh *github.Client) error {
		log := clog.FromContext(ctx).With("resource", res.String())

		// Create state for this resource
		state := githubreconciler.NewState[PullRequestState](stateManager.Identity(), gh, res)

		// Fetch current state
		_, err := state.Fetch(ctx)
		if err != nil {
			log.Errorf("Failed to fetch current state: %v", err)
			// Continue with fresh state
		}

		// Initialize new state
		newState := &PullRequestState{
			ChecksStatus: make(map[string]string),
		}

		// Fetch the pull request
		pr, _, err := gh.PullRequests.Get(ctx, res.Owner, res.Repo, res.Number)
		if err != nil {
			return fmt.Errorf("fetching pull request: %w", err)
		}

		newState.PRState = pr.GetState()
		newState.Mergeable = pr.Mergeable

		// Log PR details
		log.With(
			"title", pr.GetTitle(),
			"state", pr.GetState(),
			"author", pr.GetUser().GetLogin(),
			"mergeable", pr.Mergeable,
		).Info("Processing GitHub pull request")

		// Count commits
		commitsOpts := &github.ListOptions{PerPage: 100}
		commitCount := 0
		for {
			commits, resp, err := gh.PullRequests.ListCommits(ctx, res.Owner, res.Repo, res.Number, commitsOpts)
			if err != nil {
				return fmt.Errorf("listing commits: %w", err)
			}
			commitCount += len(commits)
			if resp.NextPage == 0 {
				break
			}
			commitsOpts.Page = resp.NextPage
		}
		newState.CommitCount = commitCount

		// Count comments
		commentOpts := &github.IssueListCommentsOptions{
			ListOptions: github.ListOptions{PerPage: 100},
		}
		commentCount := 0
		for {
			comments, resp, err := gh.Issues.ListComments(ctx, res.Owner, res.Repo, res.Number, commentOpts)
			if err != nil {
				return fmt.Errorf("listing comments: %w", err)
			}
			commentCount += len(comments)
			if resp.NextPage == 0 {
				break
			}
			commentOpts.Page = resp.NextPage
		}
		newState.CommentCount = commentCount

		// Count reviews
		reviewOpts := &github.ListOptions{PerPage: 100}
		reviewCount := 0
		approvedCount := 0
		changesRequestedCount := 0
		for {
			reviews, resp, err := gh.PullRequests.ListReviews(ctx, res.Owner, res.Repo, res.Number, reviewOpts)
			if err != nil {
				return fmt.Errorf("listing reviews: %w", err)
			}
			for _, review := range reviews {
				if review.GetState() == "APPROVED" {
					approvedCount++
				} else if review.GetState() == "CHANGES_REQUESTED" {
					changesRequestedCount++
				}
			}
			reviewCount += len(reviews)
			if resp.NextPage == 0 {
				break
			}
			reviewOpts.Page = resp.NextPage
		}
		newState.ReviewCount = reviewCount

		// Count files
		filesOpts := &github.ListOptions{PerPage: 100}
		fileCount := 0
		for {
			files, resp, err := gh.PullRequests.ListFiles(ctx, res.Owner, res.Repo, res.Number, filesOpts)
			if err != nil {
				return fmt.Errorf("listing files: %w", err)
			}
			fileCount += len(files)
			if resp.NextPage == 0 {
				break
			}
			filesOpts.Page = resp.NextPage
		}
		newState.FileCount = fileCount

		// Get check runs
		if pr.Head != nil && pr.Head.SHA != nil {
			checkOpts := &github.ListCheckRunsOptions{
				ListOptions: github.ListOptions{PerPage: 100},
			}
			for {
				checkRuns, resp, err := gh.Checks.ListCheckRunsForRef(ctx, res.Owner, res.Repo, *pr.Head.SHA, checkOpts)
				if err != nil {
					log.Errorf("Failed to list check runs: %v", err)
					break
				}
				for _, check := range checkRuns.CheckRuns {
					if check.Name != nil && check.Conclusion != nil {
						newState.ChecksStatus[*check.Name] = *check.Conclusion
					}
				}
				if resp.NextPage == 0 {
					break
				}
				checkOpts.Page = resp.NextPage
			}
		}

		// Build a status message
		var checksStatus strings.Builder
		if len(newState.ChecksStatus) > 0 {
			checksStatus.WriteString("\n\n### Check Status\n")
			for name, conclusion := range newState.ChecksStatus {
				emoji := "⏳"
				switch conclusion {
				case "success":
					emoji = "✅"
				case "failure":
					emoji = "❌"
				case "cancelled":
					emoji = "🚫"
				case "skipped":
					emoji = "⏭️"
				}
				checksStatus.WriteString(fmt.Sprintf("- %s %s: %s\n", emoji, name, conclusion))
			}
		}

		mergeableStr := "Unknown"
		if newState.Mergeable != nil {
			if *newState.Mergeable {
				mergeableStr = "✅ Yes"
			} else {
				mergeableStr = "❌ No (conflicts)"
			}
		}

		message := fmt.Sprintf(`## Pull Request Status Report

**PR #%d**: %s
**State**: %s
**Mergeable**: %s
**Author**: @%s

### Statistics
- **Commits**: %d
- **Files Changed**: %d
- **Comments**: %d
- **Reviews**: %d (Approved: %d, Changes Requested: %d)%s

This pull request has been analyzed by the GitHub reconciler.`,
			pr.GetNumber(),
			pr.GetTitle(),
			pr.GetState(),
			mergeableStr,
			pr.GetUser().GetLogin(),
			commitCount,
			fileCount,
			commentCount,
			reviewCount,
			approvedCount,
			changesRequestedCount,
			checksStatus.String())

		// Commit the new state
		if err := state.Commit(ctx, newState, message); err != nil {
			return fmt.Errorf("committing state: %w", err)
		}

		log.Info("Successfully reconciled pull request")
		return nil
	}
}
