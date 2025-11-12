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

	"github.com/chainguard-dev/clog"
	"github.com/chainguard-dev/terraform-infra-common/pkg/githubreconciler"
	internaltemplate "github.com/chainguard-dev/terraform-infra-common/pkg/githubreconciler/internal/template"
	"github.com/google/go-github/v75/github"
)

// Comparable is the interface that types must implement to be used with IssueManager.
// The Equal method determines if two instances represent the same issue based on
// identity fields (e.g., ID, unique combination of fields, etc.).
type Comparable[T any] interface {
	Equal(T) bool
}

// Option configures an IM (IssueManager).
type Option[T Comparable[T]] func(*IM[T])

// IM manages the lifecycle of GitHub Issues for a specific identity.
// It uses Go templates to generate issue titles and bodies from generic data of type T.
// IssueManager can handle multiple issues per path.
// T must implement the Comparable interface to enable matching between existing and desired issues.
type IM[T Comparable[T]] struct {
	identity         string
	titleTemplate    *template.Template
	bodyTemplate     *template.Template
	labelTemplates   []*template.Template
	templateExecutor *internaltemplate.Template[T]
	owner            string
	repo             string
	maxDesiredIssues int
}

// WithLabelTemplates sets the label templates for generating dynamic labels from issue data.
func WithLabelTemplates[T Comparable[T]](templates ...*template.Template) Option[T] {
	return func(im *IM[T]) {
		im.labelTemplates = templates
	}
}

// WithOwner overrides the GitHub owner (org or user) from the resource.
// When set, all issue operations will use this owner instead of the resource's owner.
func WithOwner[T Comparable[T]](owner string) Option[T] {
	return func(im *IM[T]) {
		im.owner = owner
	}
}

// WithRepo overrides the GitHub repository from the resource.
// When set, all issue operations will use this repo instead of the resource's repo.
func WithRepo[T Comparable[T]](repo string) Option[T] {
	return func(im *IM[T]) {
		im.repo = repo
	}
}

// WithMaxDesiredIssuesPerPath sets the maximum number of desired issues allowed per path.
// Default is 1. WARNING: High values can cause GitHub API rate limit issues.
// The default of 1 is strongly recommended. Only increase if you understand the rate limit implications.
func WithMaxDesiredIssuesPerPath[T Comparable[T]](limit int) Option[T] {
	return func(im *IM[T]) {
		im.maxDesiredIssues = limit
	}
}

// New creates a new IM with the given identity and templates.
// The templates are executed with data of type T when creating or updating issues.
// Returns an error if titleTemplate or bodyTemplate is nil.
// T must implement the Comparable interface to enable matching between existing and desired issues.
func New[T Comparable[T]](identity string, titleTemplate *template.Template, bodyTemplate *template.Template, opts ...Option[T]) (*IM[T], error) {
	if titleTemplate == nil {
		return nil, errors.New("titleTemplate cannot be nil")
	}
	if bodyTemplate == nil {
		return nil, errors.New("bodyTemplate cannot be nil")
	}

	templateExecutor, err := internaltemplate.New[T](identity, "-issue-data", "issue")
	if err != nil {
		return nil, fmt.Errorf("creating template executor: %w", err)
	}

	im := &IM[T]{
		identity:         identity,
		titleTemplate:    titleTemplate,
		bodyTemplate:     bodyTemplate,
		templateExecutor: templateExecutor,
		maxDesiredIssues: 1,
	}

	for _, opt := range opts {
		opt(im)
	}

	return im, nil
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

	// Determine which owner/repo to use
	owner := res.Owner
	repo := res.Repo
	if im.owner != "" {
		owner = im.owner
	}
	if im.repo != "" {
		repo = im.repo
	}

	// Create a label to identify issues for this path
	pathLabel := im.identity + ":" + res.Path

	// Query for existing issues with this label
	// Set a reasonable upper limit to prevent quota issues
	maxExistingIssues := 100
	var allIssues []*github.Issue
	opts := &github.IssueListByRepoOptions{
		State:  "open",
		Labels: []string{pathLabel},
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	for {
		issues, resp, err := client.Issues.ListByRepo(ctx, owner, repo, opts)
		if err != nil {
			return nil, fmt.Errorf("listing issues: %w", err)
		}

		allIssues = append(allIssues, issues...)

		// Check if we've exceeded the limit
		if len(allIssues) >= maxExistingIssues {
			return nil, fmt.Errorf("found %d or more issues with label %q, exceeding safety limit of %d", len(allIssues), pathLabel, maxExistingIssues)
		}

		if resp.NextPage == 0 {
			break
		}
		opts.ListOptions.Page = resp.NextPage
	}

	log := clog.FromContext(ctx)

	// Filter out pull requests and extract data upfront (GitHub's API returns both)
	var existingIssues []existingIssue[T]
	for _, issue := range allIssues {
		if !issue.IsPullRequest() {
			// Extract embedded data from issue body
			data, err := im.templateExecutor.Extract(issue.GetBody())
			if err != nil {
				// Skip issues with malformed data - they won't be matched anyway
				log.Warnf("Skipping issue #%d: failed to extract embedded data: %v", issue.GetNumber(), err)
				continue
			}
			existingIssues = append(existingIssues, existingIssue[T]{
				issue: issue,
				data:  data,
			})
		}
	}

	return &IssueSession[T]{
		manager:        im,
		client:         client,
		resource:       res,
		owner:          owner,
		repo:           repo,
		pathLabel:      pathLabel,
		existingIssues: existingIssues,
	}, nil
}
