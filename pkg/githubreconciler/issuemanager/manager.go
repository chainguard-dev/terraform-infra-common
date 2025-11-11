/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package issuemanager

import (
	"context"
	"errors"
	"fmt"
	"text/template"

	"github.com/chainguard-dev/terraform-infra-common/pkg/githubreconciler"
	"github.com/google/go-github/v75/github"
)

// Comparable is the interface that types must implement to be used with IssueManager.
// The Equal method determines if two instances represent the same issue based on
// identity fields (e.g., ID, unique combination of fields, etc.).
type Comparable[T any] interface {
	Equal(T) bool
}

// IM manages the lifecycle of GitHub Issues for a specific identity.
// It uses Go templates to generate issue titles and bodies from generic data of type T.
// Unlike ChangeManager which handles one PR per path, IssueManager handles multiple issues per path.
// T must implement the Comparable interface to enable matching between existing and desired issues.
type IM[T Comparable[T]] struct {
	identity       string
	titleTemplate  *template.Template
	bodyTemplate   *template.Template
	labelTemplates []*template.Template
}

// New creates a new IM with the given identity and templates.
// The templates are executed with data of type T when creating or updating issues.
// Returns an error if titleTemplate or bodyTemplate is nil.
// labelTemplates are optional and will be executed with each issue's data to generate additional labels.
// T must implement the Comparable interface to enable matching between existing and desired issues.
func New[T Comparable[T]](identity string, titleTemplate *template.Template, bodyTemplate *template.Template, labelTemplates ...*template.Template) (*IM[T], error) {
	if titleTemplate == nil {
		return nil, errors.New("titleTemplate cannot be nil")
	}
	if bodyTemplate == nil {
		return nil, errors.New("bodyTemplate cannot be nil")
	}

	return &IM[T]{
		identity:       identity,
		titleTemplate:  titleTemplate,
		bodyTemplate:   bodyTemplate,
		labelTemplates: labelTemplates,
	}, nil
}

// NewSession creates a new IssueSession for the given resource.
// It validates that the resource is a Path type and queries for any existing issues
// with a label matching {identity}:{path}.
func (im *IM[T]) NewSession(
	ctx context.Context,
	client *github.Client,
	res *githubreconciler.Resource,
) (*IssueSession[T], error) {
	if res.Type != githubreconciler.ResourceTypePath {
		return nil, fmt.Errorf("issue manager only supports Path resources, got: %v", res.Type)
	}

	// Create a label to identify issues for this path
	pathLabel := im.identity + ":" + res.Path

	// Query for existing issues with this label
	issues, _, err := client.Issues.ListByRepo(ctx, res.Owner, res.Repo, &github.IssueListByRepoOptions{
		State:  "open",
		Labels: []string{pathLabel},
		ListOptions: github.ListOptions{
			PerPage: 100, // Handle up to 100 issues per path
		},
	})
	if err != nil {
		return nil, fmt.Errorf("listing issues: %w", err)
	}

	// Filter out pull requests (GitHub's API returns both)
	var existingIssues []*github.Issue
	for _, issue := range issues {
		if issue.PullRequestLinks == nil {
			existingIssues = append(existingIssues, issue)
		}
	}

	return &IssueSession[T]{
		manager:        im,
		client:         client,
		resource:       res,
		pathLabel:      pathLabel,
		existingIssues: existingIssues,
	}, nil
}
