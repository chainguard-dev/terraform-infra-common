/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package changemanager

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/chainguard-dev/clog"
	"github.com/chainguard-dev/terraform-infra-common/pkg/githubreconciler"
	"github.com/chainguard-dev/terraform-infra-common/pkg/workqueue"
	"github.com/google/go-github/v75/github"
)

// Session represents work on a specific PR for a specific resource path.
type Session[T any] struct {
	manager    *CM[T]
	client     *github.Client
	resource   *githubreconciler.Resource
	owner      string
	repo       string
	branchName string
	existingPR *github.PullRequest
}

// HasSkipLabel checks if the existing PR has a skip label.
// Returns false if no existing PR exists.
func (s *Session[T]) HasSkipLabel() bool {
	if s.existingPR == nil {
		return false
	}

	skipLabel := "skip:" + s.manager.identity
	for _, label := range s.existingPR.Labels {
		if label.GetName() == skipLabel {
			return true
		}
	}
	return false
}

// CloseAnyOutstanding closes the existing PR if one exists.
// If message is non-empty, it posts the message as a comment before closing.
// This is a no-op if no PR exists.
func (s *Session[T]) CloseAnyOutstanding(ctx context.Context, message string) error {
	if s.existingPR == nil {
		return nil
	}

	log := clog.FromContext(ctx)
	log.Infof("Closing PR #%d", s.existingPR.GetNumber())

	// Post message as a comment if provided
	if message != "" {
		if _, _, err := s.client.Issues.CreateComment(ctx, s.owner, s.repo, s.existingPR.GetNumber(), &github.IssueComment{
			Body: github.Ptr(message),
		}); err != nil {
			return fmt.Errorf("posting comment: %w", err)
		}
	}

	_, _, err := s.client.PullRequests.Edit(ctx, s.owner, s.repo, s.existingPR.GetNumber(), &github.PullRequest{
		State: github.Ptr("closed"),
	})
	if err != nil {
		return fmt.Errorf("closing pull request: %w", err)
	}

	return nil
}

// Upsert creates a new PR or updates an existing one with the provided properties.
// It only calls makeChanges when refresh is needed or when creating a new PR.
// Returns a RequeueAfter error if GitHub is still computing the PR's mergeable status.
func (s *Session[T]) Upsert(
	ctx context.Context,
	data *T,
	draft bool,
	labels []string,
	makeChanges func(ctx context.Context, branchName string) error,
) (prURL string, err error) {
	log := clog.FromContext(ctx)

	// Check if refresh is needed
	needsRefresh, err := s.needsRefresh(ctx, data)
	if err != nil {
		return "", err
	}
	if !needsRefresh {
		log.Info("PR is up to date, no refresh needed")
		return s.existingPR.GetHTMLURL(), nil
	}

	// Make code changes on the branch
	if err := makeChanges(ctx, s.branchName); err != nil {
		return "", fmt.Errorf("making changes: %w", err)
	}

	// Generate PR title and body from templates
	title, err := s.manager.templateExecutor.Execute(s.manager.titleTemplate, data)
	if err != nil {
		return "", fmt.Errorf("executing title template: %w", err)
	}

	body, err := s.manager.templateExecutor.Execute(s.manager.bodyTemplate, data)
	if err != nil {
		return "", fmt.Errorf("executing body template: %w", err)
	}

	// Embed data in body
	body, err = s.manager.templateExecutor.Embed(body, data)
	if err != nil {
		return "", fmt.Errorf("embedding data: %w", err)
	}

	if s.existingPR == nil {
		// Create new PR
		log.Infof("Creating new PR with head %s and base %s", s.branchName, s.resource.Ref)

		pr, _, err := s.client.PullRequests.Create(ctx, s.owner, s.repo, &github.NewPullRequest{
			Title: github.Ptr(title),
			Body:  github.Ptr(body),
			Head:  github.Ptr(s.branchName),
			Base:  github.Ptr(s.resource.Ref),
			Draft: github.Ptr(draft),
		})
		if err != nil {
			return "", fmt.Errorf("creating pull request: %w", err)
		}

		// Apply labels
		if len(labels) > 0 {
			if _, _, err := s.client.Issues.AddLabelsToIssue(ctx, s.owner, s.repo, pr.GetNumber(), labels); err != nil {
				return "", fmt.Errorf("adding labels: %w", err)
			}
		}

		log.Infof("Created PR #%d: %s", pr.GetNumber(), pr.GetHTMLURL())
		return pr.GetHTMLURL(), nil
	}

	// Update existing PR
	log.Infof("Updating existing PR #%d", s.existingPR.GetNumber())

	_, _, err = s.client.PullRequests.Edit(ctx, s.owner, s.repo, s.existingPR.GetNumber(), &github.PullRequest{
		Title: github.Ptr(title),
		Body:  github.Ptr(body),
		Draft: github.Ptr(draft),
	})
	if err != nil {
		return "", fmt.Errorf("updating pull request: %w", err)
	}

	// Replace labels
	if _, _, err := s.client.Issues.ReplaceLabelsForIssue(ctx, s.owner, s.repo, s.existingPR.GetNumber(), labels); err != nil {
		return "", fmt.Errorf("replacing labels: %w", err)
	}

	log.Infof("Updated PR #%d: %s", s.existingPR.GetNumber(), s.existingPR.GetHTMLURL())
	return s.existingPR.GetHTMLURL(), nil
}

// needsRefresh determines if an existing PR needs to be refreshed.
// Returns true if no existing PR, PR has merge conflict, or embedded data differs.
// Returns an error if the Mergeable status is still being computed by GitHub (RequeueAfter 5 minutes).
func (s *Session[T]) needsRefresh(ctx context.Context, expected *T) (bool, error) {
	if s.existingPR == nil {
		return true, nil
	}

	log := clog.FromContext(ctx)

	// Check if GitHub is still computing the mergeable status
	// See: https://docs.github.com/en/rest/pulls/pulls?apiVersion=2022-11-28#get-a-pull-request
	// "The value of the mergeable attribute can be true, false, or null. If the value is null,
	// then GitHub has started a background job to compute the mergeability."
	if s.existingPR.Mergeable == nil {
		log.Info("PR mergeable status is still being computed by GitHub, requeueing")
		return false, workqueue.RequeueAfter(5 * time.Minute)
	}

	// Check for merge conflicts
	if !*s.existingPR.Mergeable {
		log.Info("PR has merge conflict, refresh needed")
		return true, nil
	}

	// Extract embedded data from PR body
	existing, err := s.manager.templateExecutor.Extract(s.existingPR.GetBody())
	if err != nil {
		log.Warnf("Failed to extract data from PR body: %v", err)
		return true, nil
	}

	// Compare data using deep equality
	if !reflect.DeepEqual(existing, expected) {
		log.Info("PR data differs, refresh needed")
		return true, nil
	}

	return false, nil
}
