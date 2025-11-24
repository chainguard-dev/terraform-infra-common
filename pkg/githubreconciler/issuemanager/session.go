/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package issuemanager

import (
	"context"
	"fmt"
	"reflect"

	"github.com/chainguard-dev/clog"
	"github.com/chainguard-dev/terraform-infra-common/pkg/githubreconciler"
	"github.com/google/go-github/v75/github"
)

// existingIssue represents an existing GitHub issue along with its extracted data.
// This avoids the need to extract data multiple times during reconciliation.
type existingIssue[T Comparable[T]] struct {
	issue *github.Issue
	data  *T
}

// IssueSession represents work on multiple issues for a specific resource path.
// Unlike Session in changemanager which handles a single PR, IssueSession manages multiple issues.
// T must implement the Comparable interface to enable matching between existing and desired issues.
type IssueSession[T Comparable[T]] struct {
	manager        *IM[T]
	client         *github.Client
	resource       *githubreconciler.Resource
	owner          string
	repo           string
	pathLabel      string
	existingIssues []existingIssue[T]
}

// hasSkipLabel checks if a specific issue has the skip label.
// Returns true if the issue has a label matching "skip:{identity}".
func (s *IssueSession[T]) hasSkipLabel(issue *github.Issue) bool {
	skipLabel := "skip:" + s.manager.identity
	for _, label := range issue.Labels {
		if label.GetName() == skipLabel {
			return true
		}
	}
	return false
}

// Reconcile reconciles the issue state with the desired state by creating, updating, and closing issues.
// It performs a complete reconciliation:
// - Creates new issues for desired states without matching existing issues
// - Updates existing issues that match desired states (if content changed)
// - Closes existing issues that don't match any desired state
// Issues with the skip label are preserved and not modified in any phase.
// The pathLabel is automatically added to the provided labels.
// Returns a slice of issue URLs in the same order as the input data.
func (s *IssueSession[T]) Reconcile(
	ctx context.Context,
	desired []*T,
	labels []string,
	closeMessage string,
) ([]string, error) {
	log := clog.FromContext(ctx)

	// Check for duplicate desired states
	for i := range desired {
		for j := i + 1; j < len(desired); j++ {
			if (*desired[i]).Equal(*desired[j]) {
				return nil, fmt.Errorf("duplicate desired state detected: entries at index %d and %d are equivalent", i, j)
			}
		}
	}

	// Check if desired issues exceed the limit
	if len(desired) > s.manager.maxDesiredIssues {
		return nil, fmt.Errorf("desired issues (%d) exceeds limit (%d) for path %s", len(desired), s.manager.maxDesiredIssues, s.resource.Path)
	}

	// Add pathLabel to labels
	allLabels := make([]string, 0, 1+len(labels))
	allLabels = append(allLabels, s.pathLabel)
	allLabels = append(allLabels, labels...)

	issueURLs := make([]string, len(desired))

	// Track which existing issues matched the desired state
	matchedIssues := make(map[int]struct{})

	// Phase 1: Create or update issues for desired state
	for i, data := range desired {
		// Try to find matching existing issue
		existing := s.findMatchingIssue(data)

		if existing != nil {
			// Mark this issue as matched
			matchedIssues[existing.issue.GetNumber()] = struct{}{}

			// Check if issue has skip label
			if s.hasSkipLabel(existing.issue) {
				log.Infof("Issue #%d has skip label, skipping update", existing.issue.GetNumber())
				issueURLs[i] = existing.issue.GetHTMLURL()
				continue
			}

			// Check if update is needed
			if !s.needsUpdate(ctx, existing, data) {
				log.Infof("Issue #%d is up to date, no refresh needed", existing.issue.GetNumber())
				issueURLs[i] = existing.issue.GetHTMLURL()
				continue
			}

			// Update existing issue
			url, err := s.updateIssue(ctx, existing.issue, data, allLabels)
			if err != nil {
				return nil, fmt.Errorf("updating issue #%d: %w", existing.issue.GetNumber(), err)
			}
			issueURLs[i] = url
		} else {
			// Create new issue
			url, err := s.createIssue(ctx, data, allLabels)
			if err != nil {
				return nil, fmt.Errorf("creating issue: %w", err)
			}
			issueURLs[i] = url
		}
	}

	// Phase 2: Close any unmatched existing issues
	for _, existing := range s.existingIssues {
		// Skip if this issue matched desired state
		if _, matched := matchedIssues[existing.issue.GetNumber()]; matched {
			continue
		}

		// Check if issue has skip label
		if s.hasSkipLabel(existing.issue) {
			log.Infof("Issue #%d has skip label, preserving issue", existing.issue.GetNumber())
			continue
		}

		// Close the issue
		log.Infof("Closing issue #%d as it's not in the desired set", existing.issue.GetNumber())

		// Post message as a comment if provided
		if closeMessage != "" {
			if _, _, err := s.client.Issues.CreateComment(ctx, s.owner, s.repo, existing.issue.GetNumber(), &github.IssueComment{
				Body: github.Ptr(closeMessage),
			}); err != nil {
				return nil, fmt.Errorf("posting comment on issue #%d: %w", existing.issue.GetNumber(), err)
			}
		}

		// Close the issue
		if _, _, err := s.client.Issues.Edit(ctx, s.owner, s.repo, existing.issue.GetNumber(), &github.IssueRequest{
			State: github.Ptr("closed"),
		}); err != nil {
			return nil, fmt.Errorf("closing issue #%d: %w", existing.issue.GetNumber(), err)
		}

		log.Infof("Closed issue #%d", existing.issue.GetNumber())
	}

	return issueURLs, nil
}

// findMatchingIssue finds an existing issue that matches the given data using the Equal method.
// Returns nil if no match is found.
func (s *IssueSession[T]) findMatchingIssue(data *T) *existingIssue[T] {
	for _, existing := range s.existingIssues {
		if (*existing.data).Equal(*data) {
			return &existing
		}
	}

	return nil
}

// needsUpdate determines if an existing issue needs to be updated.
// Returns true if the embedded data differs from expected.
func (s *IssueSession[T]) needsUpdate(ctx context.Context, existing *existingIssue[T], expected *T) bool {
	log := clog.FromContext(ctx)

	// Compare data for equality
	if !reflect.DeepEqual(*existing.data, *expected) {
		log.Infof("Issue #%d data differs, update needed", existing.issue.GetNumber())
		return true
	}

	return false
}

// generateLabels generates labels from label templates by executing them with the provided data.
// Returns an empty slice if there are no label templates or if all templates fail to execute.
func (s *IssueSession[T]) generateLabels(ctx context.Context, data *T) []string {
	if len(s.manager.labelTemplates) == 0 {
		return nil
	}

	log := clog.FromContext(ctx)
	labels := make([]string, 0, len(s.manager.labelTemplates))

	for _, tmpl := range s.manager.labelTemplates {
		label, err := s.manager.templateExecutor.Execute(tmpl, data)
		if err != nil {
			log.Warnf("Failed to execute label template %q: %v", tmpl.Name(), err)
			continue
		}
		if label != "" {
			labels = append(labels, label)
		}
	}

	return labels
}

// prepareIssueRequest generates the title, body, and labels for an issue from the provided data.
// This is used by both createIssue and updateIssue to avoid code duplication.
func (s *IssueSession[T]) prepareIssueRequest(ctx context.Context, data *T, labels []string) (string, string, []string, error) {
	// Generate issue title and body from templates
	title, err := s.manager.templateExecutor.Execute(s.manager.titleTemplate, data)
	if err != nil {
		return "", "", nil, fmt.Errorf("executing title template: %w", err)
	}

	body, err := s.manager.templateExecutor.Execute(s.manager.bodyTemplate, data)
	if err != nil {
		return "", "", nil, fmt.Errorf("executing body template: %w", err)
	}

	// Embed data in body
	body, err = s.manager.templateExecutor.Embed(body, data)
	if err != nil {
		return "", "", nil, fmt.Errorf("embedding data: %w", err)
	}

	// Generate labels from templates and merge with static labels
	generatedLabels := s.generateLabels(ctx, data)
	allLabels := make([]string, 0, len(labels)+len(generatedLabels))
	allLabels = append(allLabels, labels...)
	allLabels = append(allLabels, generatedLabels...)

	return title, body, allLabels, nil
}

// createIssue creates a new issue with the provided data and labels.
func (s *IssueSession[T]) createIssue(ctx context.Context, data *T, labels []string) (string, error) {
	log := clog.FromContext(ctx)

	title, body, allLabels, err := s.prepareIssueRequest(ctx, data, labels)
	if err != nil {
		return "", err
	}

	log.Info("Creating new issue")

	issue, _, err := s.client.Issues.Create(ctx, s.owner, s.repo, &github.IssueRequest{
		Title:  github.Ptr(title),
		Body:   github.Ptr(body),
		Labels: &allLabels,
	})
	if err != nil {
		return "", fmt.Errorf("creating issue: %w", err)
	}

	log.Infof("Created issue #%d: %s", issue.GetNumber(), issue.GetHTMLURL())
	return issue.GetHTMLURL(), nil
}

// updateIssue updates an existing issue with the provided data and labels.
func (s *IssueSession[T]) updateIssue(ctx context.Context, issue *github.Issue, data *T, labels []string) (string, error) {
	log := clog.FromContext(ctx)

	title, body, allLabels, err := s.prepareIssueRequest(ctx, data, labels)
	if err != nil {
		return "", err
	}

	log.Infof("Updating existing issue #%d", issue.GetNumber())

	updated, _, err := s.client.Issues.Edit(ctx, s.owner, s.repo, issue.GetNumber(), &github.IssueRequest{
		Title:  github.Ptr(title),
		Body:   github.Ptr(body),
		Labels: &allLabels,
	})
	if err != nil {
		return "", fmt.Errorf("updating issue: %w", err)
	}

	log.Infof("Updated issue #%d: %s", issue.GetNumber(), updated.GetHTMLURL())
	return updated.GetHTMLURL(), nil
}
