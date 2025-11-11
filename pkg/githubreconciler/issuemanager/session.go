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

// IssueSession represents work on multiple issues for a specific resource path.
// Unlike Session in changemanager which handles a single PR, IssueSession manages multiple issues.
// T must implement the Comparable interface to enable matching between existing and desired issues.
type IssueSession[T Comparable[T]] struct {
	manager        *IM[T]
	client         *github.Client
	resource       *githubreconciler.Resource
	pathLabel      string
	existingIssues []*github.Issue
}

// HasSkipLabel checks if any existing issue has a skip label.
// Returns false if no existing issues exist.
func (s *IssueSession[T]) HasSkipLabel() bool {
	if len(s.existingIssues) == 0 {
		return false
	}

	skipLabel := "skip:" + s.manager.identity
	for _, issue := range s.existingIssues {
		for _, label := range issue.Labels {
			if label.GetName() == skipLabel {
				return true
			}
		}
	}
	return false
}

// UpsertMany creates or updates multiple issues based on the provided data.
// It uses the Equal method on T to determine which existing issues correspond to which desired data.
// The pathLabel is automatically added to the provided labels.
// Returns a slice of issue URLs in the same order as the input data.
func (s *IssueSession[T]) UpsertMany(
	ctx context.Context,
	desired []*T,
	labels []string,
) ([]string, error) {
	log := clog.FromContext(ctx)

	// Add pathLabel to labels
	allLabels := append([]string{s.pathLabel}, labels...)

	issueURLs := make([]string, len(desired))

	for i, data := range desired {
		// Try to find matching existing issue
		existingIssue := s.findMatchingIssue(ctx, data)

		if existingIssue != nil {
			// Check if update is needed
			needsUpdate, err := s.needsUpdate(ctx, existingIssue, data)
			if err != nil {
				return nil, fmt.Errorf("checking if issue #%d needs update: %w", existingIssue.GetNumber(), err)
			}

			if !needsUpdate {
				log.Infof("Issue #%d is up to date, no refresh needed", existingIssue.GetNumber())
				issueURLs[i] = existingIssue.GetHTMLURL()
				continue
			}

			// Update existing issue
			url, err := s.updateIssue(ctx, existingIssue, data, allLabels)
			if err != nil {
				return nil, fmt.Errorf("updating issue #%d: %w", existingIssue.GetNumber(), err)
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

	return issueURLs, nil
}

// CloseAnyOutstanding closes any existing issues that don't match the desired data set.
// It uses the Equal method on T to determine which issues should be kept.
// If message is non-empty, it posts the message as a comment before closing.
func (s *IssueSession[T]) CloseAnyOutstanding(
	ctx context.Context,
	desired []*T,
	message string,
) error {
	log := clog.FromContext(ctx)

	for _, issue := range s.existingIssues {
		// Extract embedded data from issue
		existing, err := s.manager.extractData(issue.GetBody())
		if err != nil {
			log.Warnf("Failed to extract data from issue #%d body, skipping: %v", issue.GetNumber(), err)
			continue
		}

		// Check if this issue matches any desired data
		matched := false
		for _, data := range desired {
			if (*existing).Equal(*data) {
				matched = true
				break
			}
		}

		if !matched {
			// Close the issue
			log.Infof("Closing issue #%d as it's not in the desired set", issue.GetNumber())

			// Post message as a comment if provided
			if message != "" {
				if _, _, err := s.client.Issues.CreateComment(ctx, s.resource.Owner, s.resource.Repo, issue.GetNumber(), &github.IssueComment{
					Body: github.Ptr(message),
				}); err != nil {
					return fmt.Errorf("posting comment on issue #%d: %w", issue.GetNumber(), err)
				}
			}

			// Close the issue
			if _, _, err := s.client.Issues.Edit(ctx, s.resource.Owner, s.resource.Repo, issue.GetNumber(), &github.IssueRequest{
				State: github.Ptr("closed"),
			}); err != nil {
				return fmt.Errorf("closing issue #%d: %w", issue.GetNumber(), err)
			}

			log.Infof("Closed issue #%d", issue.GetNumber())
		}
	}

	return nil
}

// findMatchingIssue finds an existing issue that matches the given data using the Equal method.
// Returns nil if no match is found.
func (s *IssueSession[T]) findMatchingIssue(ctx context.Context, data *T) *github.Issue {
	log := clog.FromContext(ctx)

	for _, issue := range s.existingIssues {
		existing, err := s.manager.extractData(issue.GetBody())
		if err != nil {
			log.Warnf("Failed to extract data from issue #%d body, skipping: %v", issue.GetNumber(), err)
			continue
		}

		if (*existing).Equal(*data) {
			return issue
		}
	}

	return nil
}

// needsUpdate determines if an existing issue needs to be updated.
// Returns true if the embedded data differs from expected.
func (s *IssueSession[T]) needsUpdate(ctx context.Context, issue *github.Issue, expected *T) (bool, error) {
	log := clog.FromContext(ctx)

	// Extract embedded data from issue body
	existing, err := s.manager.extractData(issue.GetBody())
	if err != nil {
		log.Warnf("Failed to extract data from issue #%d body: %v", issue.GetNumber(), err)
		return true, nil
	}

	// Compare data using deep equality
	if !reflect.DeepEqual(existing, expected) {
		log.Infof("Issue #%d data differs, update needed", issue.GetNumber())
		return true, nil
	}

	return false, nil
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
		label, err := s.manager.executeTemplate(tmpl, data)
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

// createIssue creates a new issue with the provided data and labels.
func (s *IssueSession[T]) createIssue(ctx context.Context, data *T, labels []string) (string, error) {
	log := clog.FromContext(ctx)

	// Generate issue title and body from templates
	title, err := s.manager.executeTemplate(s.manager.titleTemplate, data)
	if err != nil {
		return "", fmt.Errorf("executing title template: %w", err)
	}

	body, err := s.manager.executeTemplate(s.manager.bodyTemplate, data)
	if err != nil {
		return "", fmt.Errorf("executing body template: %w", err)
	}

	// Embed data in body
	body, err = s.manager.embedData(body, data)
	if err != nil {
		return "", fmt.Errorf("embedding data: %w", err)
	}

	// Generate labels from templates and merge with static labels
	generatedLabels := s.generateLabels(ctx, data)
	labels = append(labels, generatedLabels...)

	log.Info("Creating new issue")

	issue, _, err := s.client.Issues.Create(ctx, s.resource.Owner, s.resource.Repo, &github.IssueRequest{
		Title:  github.Ptr(title),
		Body:   github.Ptr(body),
		Labels: &labels,
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

	// Generate issue title and body from templates
	title, err := s.manager.executeTemplate(s.manager.titleTemplate, data)
	if err != nil {
		return "", fmt.Errorf("executing title template: %w", err)
	}

	body, err := s.manager.executeTemplate(s.manager.bodyTemplate, data)
	if err != nil {
		return "", fmt.Errorf("executing body template: %w", err)
	}

	// Embed data in body
	body, err = s.manager.embedData(body, data)
	if err != nil {
		return "", fmt.Errorf("embedding data: %w", err)
	}

	log.Infof("Updating existing issue #%d", issue.GetNumber())

	updated, _, err := s.client.Issues.Edit(ctx, s.resource.Owner, s.resource.Repo, issue.GetNumber(), &github.IssueRequest{
		Title: github.Ptr(title),
		Body:  github.Ptr(body),
	})
	if err != nil {
		return "", fmt.Errorf("updating issue: %w", err)
	}

	// Generate labels from templates and merge with static labels
	generatedLabels := s.generateLabels(ctx, data)
	labels = append(labels, generatedLabels...)

	// Replace labels
	if _, _, err := s.client.Issues.ReplaceLabelsForIssue(ctx, s.resource.Owner, s.resource.Repo, issue.GetNumber(), labels); err != nil {
		return "", fmt.Errorf("replacing labels: %w", err)
	}

	log.Infof("Updated issue #%d: %s", issue.GetNumber(), updated.GetHTMLURL())
	return updated.GetHTMLURL(), nil
}
